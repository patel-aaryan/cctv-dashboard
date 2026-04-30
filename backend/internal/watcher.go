package internal

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

const filenameTimeLayout = "2006-01-02_15-04-05"

type Watcher struct {
	root string
	db   *sql.DB
	fs   *fsnotify.Watcher
}

func NewWatcher(root string, db *sql.DB) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{root: root, db: db, fs: fw}
	if err := w.addRecursive(root); err != nil {
		fw.Close()
		return nil, fmt.Errorf("add watch %s: %w", root, err)
	}
	return w, nil
}

func (w *Watcher) Close() error {
	return w.fs.Close()
}

func (w *Watcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return w.fs.Add(path)
		}
		return nil
	})
}

func (w *Watcher) Run() {
	for {
		select {
		case event, ok := <-w.fs.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-w.fs.Errors:
			if !ok {
				return
			}
			slog.Warn("watcher error", "err", err)
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	if event.Has(fsnotify.Create) {
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			if err := w.addRecursive(event.Name); err != nil {
				slog.Warn("watch new dir", "path", event.Name, "err", err)
			}
			return
		}
	}
	if !isMP4(event.Name) {
		return
	}
	switch {
	case event.Has(fsnotify.Create):
		w.indexFile(event.Name)
	case event.Has(fsnotify.Remove), event.Has(fsnotify.Rename):
		if err := deleteVideoByPath(w.db, event.Name); err != nil {
			slog.Warn("delete video", "path", event.Name, "err", err)
		}
	}
}

func (w *Watcher) indexFile(path string) {
	v, err := parseFilename(path)
	if err != nil {
		slog.Debug("skip non-conforming filename", "path", path, "err", err)
		return
	}
	inserted, err := insertVideo(w.db, v)
	if err != nil {
		slog.Warn("insert video", "path", path, "err", err)
		return
	}
	if inserted {
		slog.Info("indexed video", "camera", v.CameraName, "ts", v.Timestamp.Format(time.RFC3339), "path", path)
	}
}

func InitialScan(root string, db *sql.DB) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !isMP4(d.Name()) {
			return nil
		}
		count++
		v, err := parseFilename(path)
		if err != nil {
			slog.Debug("scan: skip non-conforming filename", "path", path, "err", err)
			return nil
		}
		if _, err := insertVideo(db, v); err != nil {
			slog.Warn("scan: insert", "path", path, "err", err)
		}
		return nil
	})
	return count, err
}

func isMP4(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".mp4")
}

func parseFilename(path string) (Video, error) {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	parts := strings.Split(base, "_")
	if len(parts) < 3 {
		return Video{}, errors.New("expected camName_YYYY-MM-DD_HH-MM-SS.mp4")
	}
	tsStr := parts[len(parts)-2] + "_" + parts[len(parts)-1]
	camName := strings.Join(parts[:len(parts)-2], "_")
	if camName == "" {
		return Video{}, errors.New("missing camera name")
	}
	ts, err := time.ParseInLocation(filenameTimeLayout, tsStr, time.Local)
	if err != nil {
		return Video{}, fmt.Errorf("parse timestamp %q: %w", tsStr, err)
	}
	return Video{CameraName: camName, Timestamp: ts, Filepath: path}, nil
}
