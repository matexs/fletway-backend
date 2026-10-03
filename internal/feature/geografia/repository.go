package geografia

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/matexs/fletway-backend/internal/platform/database"
)

var (
	errSinTransportista = errors.New("geografia: la cuenta no es de Transportista")
	errZonaInexistente  = errors.New("geografia: zona inexistente")
)

type repository struct {
	db *database.DB
}

func (r *repository) zonas(ctx context.Context, id database.Identity) ([]ZonaResponse, error) {
	var out []ZonaResponse
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT id::text, nombre, provincia FROM zona ORDER BY provincia, nombre`)
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(rows, pgx.RowToStructByPos[ZonaResponse])
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listar zonas: %w", err)
	}
	return out, nil
}

// deTransportista lee las zonas del Transportista. Devuelve errSinTransportista.
func (r *repository) deTransportista(ctx context.Context, id database.Identity) ([]string, error) {
	ids := []string{}
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		if err := exigirTransportista(ctx, tx); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT zona_id::text FROM transportista_zona
			WHERE transportista_id = (select auth.uid()) ORDER BY zona_id`)
		if err != nil {
			return err
		}
		leidos, err := pgx.CollectRows(rows, pgx.RowTo[string])
		ids = append(ids, leidos...)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("leer zonas del transportista: %w", err)
	}
	return ids, nil
}

// reemplazar deja exactamente zonaIDs como zonas del Transportista, en una
// transacción. Devuelve errSinTransportista o errZonaInexistente.
func (r *repository) reemplazar(ctx context.Context, id database.Identity, zonaIDs []string) error {
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		if err := exigirTransportista(ctx, tx); err != nil {
			return err
		}
		var existentes int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM zona WHERE id = ANY($1::uuid[])`, zonaIDs).
			Scan(&existentes); err != nil {
			return err
		}
		if existentes != len(zonaIDs) {
			return errZonaInexistente
		}
		if _, err := tx.Exec(ctx, `DELETE FROM transportista_zona
			WHERE transportista_id = (select auth.uid()) AND NOT (zona_id = ANY($1::uuid[]))`, zonaIDs); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO transportista_zona (transportista_id, zona_id)
			SELECT (select auth.uid()), z FROM unnest($1::uuid[]) AS z
			ON CONFLICT DO NOTHING`, zonaIDs)
		return err
	})
	if err != nil {
		return fmt.Errorf("reemplazar zonas: %w", err)
	}
	return nil
}

func exigirTransportista(ctx context.Context, tx pgx.Tx) error {
	var es bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM transportista WHERE usuario_id = (select auth.uid()))`).Scan(&es); err != nil {
		return err
	}
	if !es {
		return errSinTransportista
	}
	return nil
}
