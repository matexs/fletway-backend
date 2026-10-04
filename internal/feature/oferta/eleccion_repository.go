package oferta

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/matexs/fletway-backend/internal/platform/database"
)

var (
	errOfertaNoPendiente    = errors.New("oferta: no está pendiente")
	errSolicitudNoAsignable = errors.New("oferta: la solicitud ya no está publicada")
	errTransportistaBaja    = errors.New("oferta: el Transportista no puede tomar el viaje")
)

// ofertasDeSolicitud lee las ofertas pendientes de una solicitud del Cliente de
// la identidad y los ayudantes que pidió. Devuelve errSolicitudInexistente si
// la solicitud no existe o no es suya.
func (r *repository) ofertasDeSolicitud(ctx context.Context, id database.Identity, solicitudID string) (int, []OfertaParaCliente, error) {
	var pedidos int
	out := []OfertaParaCliente{}
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT cantidad_ayudantes_solicitados FROM solicitud
			WHERE id = $1 AND cliente_id = (select auth.uid())`, solicitudID).Scan(&pedidos)
		if errors.Is(err, pgx.ErrNoRows) {
			return errSolicitudInexistente
		}
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			SELECT o.id::text, o.transportista_id::text, u.nombre_completo,
			       t.calificacion_promedio::float8,
			       (SELECT count(*)::int FROM resena r WHERE r.transportista_id = o.transportista_id),
			       COALESCE(t.tasa_cumplimiento, 100)::float8,
			       tv.nombre, o.cantidad_viajes, o.cantidad_ayudantes, o.precio_calculado::text, o.creado_en
			FROM oferta o
			JOIN transportista t ON t.usuario_id = o.transportista_id
			JOIN usuario u ON u.id = o.transportista_id
			JOIN vehiculo v ON v.id = o.vehiculo_id
			JOIN tipo_vehiculo tv ON tv.id = v.tipo_vehiculo_id
			WHERE o.solicitud_id = $1 AND o.estado_codigo = 'pendiente'`, solicitudID)
		if err != nil {
			return err
		}
		lista, err := pgx.CollectRows(rows, pgx.RowToStructByPos[OfertaParaCliente])
		out = append(out, lista...)
		return err
	})
	if err != nil {
		return 0, nil, fmt.Errorf("listar ofertas de la solicitud: %w", err)
	}
	return pedidos, out, nil
}

// aceptar acepta la oferta con fn_aceptar_oferta (0018) y devuelve el viaje.
// Devuelve errNoEncontrada, errOfertaNoPendiente, errSolicitudNoAsignable o
// errTransportistaBaja.
func (r *repository) aceptar(ctx context.Context, id database.Identity, ofertaID string) (ViajeConfirmadoResponse, error) {
	var v ViajeConfirmadoResponse
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		var viajeID string
		if err := tx.QueryRow(ctx, `SELECT fn_aceptar_oferta($1)::text`, ofertaID).Scan(&viajeID); err != nil {
			return traducirAceptar(err)
		}
		return tx.QueryRow(ctx, `
			SELECT vj.id::text, vj.estado_codigo, vj.solicitud_id::text, vj.transportista_id::text,
			       vj.transportista_nombre_snapshot, vj.vehiculo_patente_snapshot, vj.vehiculo_marca_modelo_snapshot,
			       vj.monto_total_snapshot::text, vj.cantidad_viajes_snapshot, vj.cantidad_ayudantes,
			       to_char(s.fecha_servicio_deseada, 'YYYY-MM-DD'),
			       vj.origen_direccion_snapshot, vj.destino_direccion_snapshot, vj.creado_en
			FROM viaje vj JOIN solicitud s ON s.id = vj.solicitud_id
			WHERE vj.id = $1`, viajeID,
		).Scan(&v.ID, &v.Estado, &v.SolicitudID, &v.TransportistaID, &v.TransportistaNombre,
			&v.VehiculoPatente, &v.VehiculoMarcaModelo, &v.MontoTotal, &v.CantidadViajes,
			&v.CantidadAyudantes, &v.FechaServicio, &v.OrigenDireccion, &v.DestinoDireccion, &v.CreadoEn)
	})
	if err != nil {
		return ViajeConfirmadoResponse{}, fmt.Errorf("aceptar oferta: %w", err)
	}
	return v, nil
}

// traducirAceptar pasa los SQLSTATE de fn_aceptar_oferta a errores del paquete.
func traducirAceptar(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "42501":
		return errNoEncontrada
	case "FW001":
		return errOfertaNoPendiente
	case "FW002":
		return errSolicitudNoAsignable
	case "FW003":
		return errTransportistaBaja
	}
	return err
}
