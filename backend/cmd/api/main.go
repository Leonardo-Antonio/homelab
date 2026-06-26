package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"homelab/backend/internal/clipboard"
	"homelab/backend/internal/config"
	"homelab/backend/internal/database"
	"homelab/backend/internal/httpapi"
	"homelab/backend/internal/notes"
	"homelab/backend/internal/photos"
	"homelab/backend/internal/settings"
	"homelab/backend/internal/storage"
	"homelab/backend/internal/terminal"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	db, err := database.Open(ctx, cfg.DatabasePath)
	if err != nil {
		slog.Error("database startup failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	clipboardRepository := clipboard.NewRepository(db)
	clipboardService := clipboard.NewService(clipboardRepository)
	clipboardHandler := clipboard.NewHandler(clipboardService)
	photoRepository := photos.NewRepository(db)
	photoService := photos.NewService(photoRepository, cfg.PhotoStorageDir)
	photoHandler := photos.NewHandler(photoService)
	terminalHandler := terminal.NewHandler(cfg.SSH, cfg.AllowedOrigin)
	notesRepository := notes.NewRepository(db)
	notesService := notes.NewService(notesRepository)
	notesHandler := notes.NewHandler(notesService)
	storageRepository := storage.NewRepository(db)
	storageService, err := storage.NewService(storageRepository, cfg.StorageDir)
	if err != nil {
		slog.Error("storage startup failed", "error", err)
		os.Exit(1)
	}
	storageHandler := storage.NewHandler(storageService)
	settingsRepository := settings.NewRepository(db)
	settingsService := settings.NewService(settingsRepository)
	settingsHandler := settings.NewHandler(settingsService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpapi.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	clipboardHandler.Register(mux)
	photoHandler.Register(mux)
	terminalHandler.Register(mux)
	notesHandler.Register(mux)
	storageHandler.Register(mux)
	settingsHandler.Register(mux)

	handler := httpapi.Recover(httpapi.Logger(httpapi.CORS(cfg.AllowedOrigin)(mux)))
	server := &http.Server{
		Addr:         cfg.Addr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	shutdownDone := make(chan struct{})
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown failed", "error", err)
		}

		close(shutdownDone)
	}()

	slog.Info("api listening", "addr", cfg.Addr, "database", cfg.DatabasePath)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}

	<-shutdownDone
	slog.Info("api stopped")
}
