// Package database maneja el acceso a Postgres (Supabase) con RLS pass-through.
//
// Decisión D-02: el backend conecta directo a Postgres con pgx/pgxpool. En cada
// unidad de trabajo transaccional se propaga el JWT del usuario a la sesión de
// Postgres (rol authenticated + claims del JWT), de modo que las políticas RLS ya
// desplegadas siguen siendo la capa de seguridad efectiva.
package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/matexs/fletway-backend/internal/platform/config"
)

// ErrSinURL se devuelve si no hay connection string para abrir el pool.
var ErrSinURL = errors.New("database: falta DATABASE_URL")

// ErrSinIdentidad se devuelve si se pide una transacción con RLS sin los claims
// del usuario: sin ellos las policies no tendrían a quién filtrar.
var ErrSinIdentidad = errors.New("database: identidad sin claims para RLS")

// DB envuelve el pool de conexiones a Postgres.
type DB struct {
	pool *pgxpool.Pool
}

// Open crea el pool de conexiones a partir de la config y verifica que la base
// responda. Devuelve ErrSinURL si DATABASE_URL está vacía.
func Open(ctx context.Context, cfg config.Config) (*DB, error) {
	if cfg.Database.URL == "" {
		return nil, ErrSinURL
	}
	pcfg, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		return nil, fmt.Errorf("parsear DATABASE_URL: %w", err)
	}
	pcfg.MaxConns = cfg.Database.MaxConns
	pcfg.MinConns = cfg.Database.MinConns
	pcfg.MaxConnLifetime = cfg.Database.MaxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		return nil, fmt.Errorf("crear pool de conexiones: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("conectar a la base: %w", err)
	}
	return &DB{pool: pool}, nil
}

// Close libera el pool. Se llama una sola vez, al apagar el servicio.
func (db *DB) Close() {
	db.pool.Close()
}

// Ping verifica que la base responda. Lo usa /readyz.
func (db *DB) Ping(ctx context.Context) error {
	if err := db.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping a la base: %w", err)
	}
	return nil
}

// Identity son los datos del usuario autenticado que se propagan a Postgres.
type Identity struct {
	// UserID es el claim "sub": usuario.id == auth.uid().
	UserID string
	// Role es el claim "role" del JWT ("authenticated" para usuarios logueados).
	Role string
	// Email es el claim "email".
	Email string
	// RawClaims es el JSON de los claims, que se carga en request.jwt.claims
	// para que auth.uid() y las policies lo lean.
	RawClaims string
	// AccessToken es el JWT original. Sólo para llamar a otras APIs de Supabase
	// en nombre del usuario (por ejemplo, firmar URLs de Storage, D-19); nunca se
	// loguea ni se persiste.
	AccessToken string
}

// WithinTx ejecuta fn dentro de una transacción con el contexto RLS del usuario:
// cambia al rol authenticated y carga los claims del JWT, ambos con alcance de
// la transacción. Es el único punto de entrada para que un repository consulte
// la base; nunca hay que usar el pool sin este contexto (CLAUDE.md §5).
//
// Si fn devuelve error, la transacción se revierte y el error se propaga
// envuelto. Devuelve ErrSinIdentidad si id no trae claims.
func (db *DB) WithinTx(ctx context.Context, id Identity, fn func(tx pgx.Tx) error) error {
	if id.RawClaims == "" || id.UserID == "" {
		return ErrSinIdentidad
	}
	err := pgx.BeginTxFunc(ctx, db.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		// SET no admite parámetros; set_config con is_local=true tiene el mismo
		// alcance que SET LOCAL (termina con la transacción).
		if _, err := tx.Exec(ctx, "SET LOCAL ROLE authenticated"); err != nil {
			return fmt.Errorf("aplicar rol authenticated: %w", err)
		}
		if _, err := tx.Exec(ctx, "SELECT set_config('request.jwt.claims', $1, true)", id.RawClaims); err != nil {
			return fmt.Errorf("cargar claims del JWT: %w", err)
		}
		return fn(tx)
	})
	if err != nil {
		return fmt.Errorf("transacción con RLS: %w", err)
	}
	return nil
}
