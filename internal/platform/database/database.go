// Package database maneja el acceso a Postgres (Supabase) con RLS pass-through.
//
// Decisión D-02: el backend conecta directo a Postgres con pgx/pgxpool. En cada
// unidad de trabajo transaccional se propaga el JWT del usuario a la sesión de
// Postgres (SET LOCAL role authenticated + SET LOCAL request.jwt.claims), de modo
// que las políticas RLS ya desplegadas siguen siendo la capa de seguridad efectiva.
//
// NOTA: este archivo es un skeleton que compila solo con stdlib. La implementación
// real con pgx se agrega cuando `go mod tidy` pueda resolver dependencias
// (ver go.mod y docs/ESTADO_PROYECTO.md → Pendientes).
package database

import (
	"context"
	"errors"

	"github.com/fletway/fletway-backend/internal/platform/config"
)

// ErrNotImplemented se devuelve mientras el pool real no está cableado.
var ErrNotImplemented = errors.New("database: pgx todavía no cableado (ver go.mod)")

// DB envuelve el pool de conexiones. Cuando se integre pgx, el campo interno pasa
// a ser *pgxpool.Pool.
type DB struct {
	// pool *pgxpool.Pool
	cfg config.DatabaseConfig
}

// Open crea el pool de conexiones a partir de la config.
//
// Implementación futura:
//
//	pcfg, err := pgxpool.ParseConfig(cfg.Database.URL)
//	pcfg.MaxConns = cfg.Database.MaxConns
//	pcfg.MinConns = cfg.Database.MinConns
//	pcfg.MaxConnLifetime = cfg.Database.MaxConnLifetime
//	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
func Open(ctx context.Context, cfg config.Config) (*DB, error) {
	_ = ctx
	return &DB{cfg: cfg.Database}, nil
}

// Close libera el pool.
func (db *DB) Close() {
	// db.pool.Close()
}

// Ping verifica conectividad (usado por /readyz).
func (db *DB) Ping(ctx context.Context) error {
	_ = ctx
	// return db.pool.Ping(ctx)
	return nil
}

// Identity son los claims del usuario autenticado que se propagan a Postgres.
type Identity struct {
	UserID string // claim "sub" == usuario.id == auth.uid()
	Role   string // "authenticated" por defecto
	Email  string
	// RawClaims es el JSON de claims tal cual, para SET LOCAL request.jwt.claims.
	RawClaims string
}

// WithinTx ejecuta fn dentro de una transacción con el contexto RLS del usuario
// aplicado (SET LOCAL). Es el punto de entrada que TODO repository debe usar para
// que las policies filtren.
//
// Implementación futura (pgx):
//
//	return pgx.BeginTxFunc(ctx, db.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
//	    if _, err := tx.Exec(ctx, "SET LOCAL role authenticated"); err != nil { return err }
//	    if _, err := tx.Exec(ctx, "SET LOCAL request.jwt.claims = $1", id.RawClaims); err != nil { return err }
//	    return fn(tx)
//	})
func (db *DB) WithinTx(ctx context.Context, id Identity, fn func(tx any) error) error {
	_ = ctx
	_ = id
	_ = fn
	return ErrNotImplemented
}
