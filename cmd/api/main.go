// Command api es el entrypoint del backend HTTP de Fletway.
//
// Arranca la configuración, el pool de base de datos (RLS pass-through, D-02),
// el pool de workers async (RNF-02) y el servidor HTTP con graceful shutdown.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/matexs/fletway-backend/internal/platform/async"
	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/config"
	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/geocodificacion"
	"github.com/matexs/fletway-backend/internal/platform/storage"
	"github.com/matexs/fletway-backend/internal/server"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fallo fatal en el arranque", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := config.NewLogger(cfg)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Base de datos: pool pgx con RLS pass-through (D-02).
	db, err := database.Open(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	// Verificación del JWT de Supabase Auth contra el JWKS (D-04). El cache de
	// claves se refresca en segundo plano mientras ctx esté vivo.
	verifier, err := auth.NewVerifier(ctx, cfg.Supabase)
	if err != nil {
		return err
	}

	// URLs firmadas de Storage con el JWT del usuario (D-19).
	archivos, err := storage.New(cfg.Supabase)
	if err != nil {
		return err
	}

	// Coordenadas de las direcciones de las solicitudes (D-20).
	geo, err := geocodificacion.Nuevo(cfg.GeocodificadorProveedor, cfg.Env)
	if err != nil {
		return err
	}

	// Workers async para tareas en segundo plano sin bloquear el request (RNF-02 / D-06).
	jobs := async.NewPool(cfg.Async.Workers, cfg.Async.QueueSize, logger)
	jobs.Start(ctx)
	defer jobs.Stop()

	srv := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      server.New(cfg, logger, db, jobs, verifier, archivos, geo).Handler(),
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("servidor escuchando", "addr", cfg.HTTP.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("señal de apagado recibida, cerrando")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
