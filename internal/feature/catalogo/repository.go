package catalogo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/matexs/fletway-backend/internal/platform/database"
)

type repository struct {
	db *database.DB
}

// objetos lee el catálogo ordenado por nombre. volumen_estimado_m3 no se expone:
// está deprecada (derivable de las medidas).
func (r *repository) objetos(ctx context.Context, id database.Identity) ([]ObjetoResponse, error) {
	var out []ObjetoResponse
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id::text, nombre, peso_estimado_kg::text, largo_m::text, ancho_m::text, alto_m::text,
			       rotacion_horizontal, rotacion_vertical, apilable
			FROM objeto ORDER BY nombre`)
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(rows, pgx.RowToStructByPos[ObjetoResponse])
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listar catálogo de objetos: %w", err)
	}
	return out, nil
}
