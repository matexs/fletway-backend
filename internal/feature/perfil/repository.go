package perfil

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/matexs/fletway-backend/internal/platform/database"
)

var errNoEncontrado = errors.New("perfil: Transportista no encontrado")

// maxResenas es cuántas reseñas recientes trae el perfil.
const maxResenas = 20

type repository struct {
	db *database.DB
}

// perfil lee el perfil de un Transportista habilitado. Devuelve errNoEncontrado.
func (r *repository) perfil(ctx context.Context, id database.Identity, transportistaID string) (PerfilResponse, error) {
	var p PerfilResponse
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			SELECT t.usuario_id::text, u.nombre_completo, t.forma_trabajo, t.calificacion_promedio::float8,
			       (SELECT count(*)::int FROM resena r WHERE r.transportista_id = t.usuario_id),
			       COALESCE(t.tasa_cumplimiento, 100)::float8, t.creado_en,
			       COALESCE((SELECT array_agg(z.nombre ORDER BY z.nombre) FROM transportista_zona tz
			                 JOIN zona z ON z.id = tz.zona_id WHERE tz.transportista_id = t.usuario_id), '{}'),
			       COALESCE((SELECT array_agg(DISTINCT tv.nombre) FROM vehiculo v
			                 JOIN tipo_vehiculo tv ON tv.id = v.tipo_vehiculo_id
			                 WHERE v.transportista_id = t.usuario_id AND v.activo), '{}')
			FROM transportista t
			JOIN usuario u ON u.id = t.usuario_id
			WHERE t.usuario_id = $1 AND t.estado_habilitacion_codigo = 'habilitado'`, transportistaID,
		).Scan(&p.ID, &p.Nombre, &p.FormaTrabajo, &p.Calificacion, &p.CantidadResenas,
			&p.TasaCumplimiento, &p.DesdeEn, &p.Zonas, &p.TiposVehiculo)
		if errors.Is(err, pgx.ErrNoRows) {
			return errNoEncontrado
		}
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			SELECT calificacion, mensaje, nombre_cliente_snapshot, creado_en FROM resena
			WHERE transportista_id = $1 ORDER BY creado_en DESC LIMIT $2`, transportistaID, maxResenas)
		if err != nil {
			return err
		}
		p.Resenas, err = pgx.CollectRows(rows, pgx.RowToStructByPos[ResenaResponse])
		return err
	})
	if err != nil {
		return PerfilResponse{}, fmt.Errorf("leer perfil: %w", err)
	}
	return p, nil
}
