package internal

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	mediaPathPrefix     = "/media/"
	headerContentType   = "Content-Type"
)

type Server struct {
	cfg        *Config
	db         *sql.DB
	mux        *http.ServeMux
	archiveAbs string
	staticAbs  string
}

func NewServer(cfg *Config, db *sql.DB) http.Handler {
	archiveAbs, _ := filepath.Abs(cfg.ArchivePath)
	staticAbs, _ := filepath.Abs(cfg.StaticPath)

	s := &Server{
		cfg:        cfg,
		db:         db,
		mux:        http.NewServeMux(),
		archiveAbs: archiveAbs,
		staticAbs:  staticAbs,
	}
	s.routes()
	return rejectTraversal(s.withCORS(s.withLogging(s.mux)))
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /api/videos", s.handleListVideos)
	s.mux.HandleFunc("GET "+mediaPathPrefix, s.handleMedia)
	s.mux.HandleFunc("/", s.handleStatic)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	if err := s.db.Ping(); err != nil {
		http.Error(w, "db unhealthy", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set(headerContentType, "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleListVideos(w http.ResponseWriter, _ *http.Request) {
	videos, err := ListVideos(s.db, 500)
	if err != nil {
		slog.Error("list videos", "err", err)
		http.Error(w, "failed to list videos", http.StatusInternalServerError)
		return
	}

	type item struct {
		ID         int64  `json:"id"`
		CameraName string `json:"camera_name"`
		Timestamp  string `json:"timestamp"`
		MediaURL   string `json:"media_url"`
	}
	out := make([]item, 0, len(videos))
	for _, v := range videos {
		rel, err := filepath.Rel(s.archiveAbs, v.Filepath)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		out = append(out, item{
			ID:         v.ID,
			CameraName: v.CameraName,
			Timestamp:  v.Timestamp.Format(time.RFC3339),
			MediaURL:   mediaPathPrefix + filepath.ToSlash(rel),
		})
	}

	w.Header().Set(headerContentType, "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		slog.Warn("encode videos", "err", err)
	}
}

func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	rel := strings.TrimPrefix(r.URL.Path, mediaPathPrefix)
	if rel == "" || !isSafeMP4(rel) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	target := filepath.Clean(filepath.Join(s.archiveAbs, filepath.FromSlash(rel)))
	if !isUnderRoot(target, s.archiveAbs) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	info, err := os.Stat(target)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}

	w.Header().Set(headerContentType, "video/mp4")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeFile(w, r, target)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, mediaPathPrefix) {
		http.NotFound(w, r)
		return
	}

	cleaned := filepath.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
	target := filepath.Join(s.staticAbs, cleaned)
	if !isUnderRoot(target, s.staticAbs) {
		http.NotFound(w, r)
		return
	}

	if info, err := os.Stat(target); err == nil && !info.IsDir() {
		http.ServeFile(w, r, target)
		return
	}
	http.ServeFile(w, r, filepath.Join(s.staticAbs, "index.html"))
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	if s.cfg.CORSOrigin == "" {
		return next
	}
	origin := s.cfg.CORSOrigin
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", headerContentType)
		w.Header().Set("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		if rw.status >= 400 || strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/health" {
			slog.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"dur_ms", time.Since(start).Milliseconds(),
				"remote", r.RemoteAddr,
			)
		}
	})
}

func rejectTraversal(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.ToLower(r.RequestURI)
		if strings.Contains(raw, "/..") ||
			strings.Contains(raw, "%2e%2e") ||
			strings.Contains(raw, "..%2f") ||
			strings.Contains(raw, "..%5c") {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func isSafeMP4(rel string) bool {
	return strings.EqualFold(filepath.Ext(rel), ".mp4") &&
		!strings.Contains(rel, "..") &&
		!strings.HasPrefix(rel, "/")
}

func isUnderRoot(target, root string) bool {
	return target == root || strings.HasPrefix(target, root+string(filepath.Separator))
}
