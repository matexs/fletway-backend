package matchmaking

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/matexs/fletway-backend/internal/platform/database"
)

type repository struct {
	db *database.DB
}

func (r *repository) esTransportista(ctx context.Context, id database.Identity) (bool, error) {
	var es bool
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM transportista WHERE usuario_id = (select auth.uid()))`).Scan(&es)
	})
	if err != nil {
		return false, fmt.Errorf("verificar transportista: %w", err)
	}
	return es, nil
}

// compatibles lista las solicitudes compatibles con el Transportista de la
// identidad, las de fecha más cercana primero. La regla vive en la base
// (fn_solicitudes_compatibles, migración 0015).
func (r *repository) compatibles(ctx context.Context, id database.Identity) ([]SolicitudCompatible, error) {
	out := []SolicitudCompatible{}
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT s.id::text, to_char(s.fecha_servicio_deseada, 'YYYY-MM-DD'),
			       to_char(s.franja_horaria_inicio, 'HH24:MI'), to_char(s.franja_horaria_fin, 'HH24:MI'),
			       zo.nombre, zd.nombre,
			       COALESCE(sum(so.cantidad), 0)::int,
			       round(COALESCE(sum(so.peso_unitario_kg * so.cantidad), 0), 2)::text,
			       round(COALESCE(sum(so.largo_m * so.ancho_m * so.alto_m * so.cantidad), 0), 2)::text,
			       s.cantidad_ayudantes_solicitados, s.creado_en
			FROM solicitud s
			JOIN zona zo ON zo.id = s.origen_zona_id
			JOIN zona zd ON zd.id = s.destino_zona_id
			LEFT JOIN solicitud_objeto so ON so.solicitud_id = s.id
			WHERE s.id IN (SELECT fn_solicitudes_compatibles())
			GROUP BY s.id, zo.nombre, zd.nombre
			ORDER BY s.fecha_servicio_deseada, s.creado_en`)
		if err != nil {
			return err
		}
		lista, err := pgx.CollectRows(rows, pgx.RowToStructByPos[SolicitudCompatible])
		out = append(out, lista...)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listar solicitudes compatibles: %w", err)
	}
	return out, nil
}

// notificar crea las notificaciones solicitud_compatible de una solicitud del
// Cliente de la identidad; es idempotente. Devuelve cuántas creó.
func (r *repository) notificar(ctx context.Context, id database.Identity, solicitudID string) (int, error) {
	var creadas int
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT fn_notificar_solicitud_compatible($1)`, solicitudID).Scan(&creadas)
	})
	if err != nil {
		return 0, fmt.Errorf("notificar solicitud compatible: %w", err)
	}
	return creadas, nil
}
