package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cctv-dashboard/internal"
)

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe the local /health endpoint and exit (used by Docker HEALTHCHECK)")
	flag.Parse()

	if *healthcheck {
		os.Exit(runHealthcheck())
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := internal.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	slog.Info("config loaded",
		"archive", cfg.ArchivePath,
		"db", cfg.DBPath,
		"port", cfg.Port,
		"static", cfg.StaticPath,
		"cors", cfg.CORSOrigin != "",
	)

	db, err := internal.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	scanned, err := internal.InitialScan(cfg.ArchivePath, db)
	if err != nil {
		slog.Warn("initial scan had errors", "err", err)
	}
	slog.Info("initial scan complete", "files", scanned)

	w, err := internal.NewWatcher(cfg.ArchivePath, db)
	if err != nil {
		return fmt.Errorf("watcher: %w", err)
	}
	defer w.Close()
	go w.Run()

	httpHandler := internal.NewServer(cfg, db)
	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpHandler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("http: %w", err)
	case sig := <-stop:
		slog.Info("shutdown signal received", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("http shutdown", "err", err)
	}
	return nil
}

func runHealthcheck() int {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://127.0.0.1:" + port + "/health")
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck: status", resp.StatusCode)
		return 1
	}
	return 0
}
