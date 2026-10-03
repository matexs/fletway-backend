package solicitud

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/geocodificacion"
)

var (
	errNoEncontrada      = errors.New("solicitud: no encontrada")
	errNoCancelable      = errors.New("solicitud: no está publicada")
	errNoVencida         = errors.New("solicitud: no está vencida")
	errObjetoInexistente = errors.New("solicitud: objeto del catálogo inexistente")
)

// zonaInfo son los datos de una zona que necesita el alta.
type zonaInfo struct {
	Nombre    string
	Provincia string
}

type repository struct {
	db *database.DB
}

// estadoSQL calcula "vencida" al leer (D-20): publicada con la fecha ya pasada.
// $1 es la fecha de hoy en Argentina.
const estadoSQL = `CASE WHEN s.estado_codigo = 'publicada' AND s.fecha_servicio_deseada < $1::date
	THEN 'vencida' ELSE s.estado_codigo END`

func (r *repository) esCliente(ctx context.Context, id database.Identity) (bool, error) {
	var es bool
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM cliente WHERE usuario_id = (select auth.uid()))`).Scan(&es)
	})
	if err != nil {
		return false, fmt.Errorf("verificar cliente: %w", err)
	}
	return es, nil
}

// zonas lee nombre y provincia de las zonas pedidas; las que no existen no
// aparecen en el mapa.
func (r *repository) zonas(ctx context.Context, id database.Identity, ids ...string) (map[string]zonaInfo, error) {
	out := map[string]zonaInfo{}
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT id::text, nombre, provincia FROM zona WHERE id = ANY($1::uuid[])`, ids)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var zid string
			var z zonaInfo
			if err := rows.Scan(&zid, &z.Nombre, &z.Provincia); err != nil {
				return err
			}
			out[zid] = z
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("leer zonas: %w", err)
	}
	return out, nil
}

// alta son los datos listos para insertar.
type alta struct {
	Origen, Destino           PuntoRequest
	OrigenCoord, DestinoCoord geocodificacion.Punto
	Fecha                     time.Time
	FranjaInicio, FranjaFin   *string
	Ayudantes                 int
	Objetos                   []ObjetoRequest
}

// crear inserta la solicitud y sus objetos en una transacción. La base la deja
// publicada y copia los valores de los objetos del catálogo (0014). Devuelve
// errObjetoInexistente si un objeto_id no existe.
func (r *repository) crear(ctx context.Context, id database.Identity, a alta) (string, error) {
	var nuevoID string
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO solicitud (cliente_id, origen_zona_id, destino_zona_id, origen_direccion, destino_direccion,
			    origen_lat, origen_lng, destino_lat, destino_lng, cantidad_ayudantes_solicitados,
			    pisos_origen, ascensor_utilizable_origen, distancia_vehiculo_origen_m,
			    pisos_destino, ascensor_utilizable_destino, distancia_vehiculo_destino_m,
			    fecha_servicio_deseada, franja_horaria_inicio, franja_horaria_fin)
			VALUES ((select auth.uid()), $1, $2, $3, $4, $5::numeric, $6::numeric, $7::numeric, $8::numeric, $9,
			        $10, $11, $12::numeric, $13, $14, $15::numeric, $16::date, $17::time, $18::time)
			RETURNING id::text`,
			a.Origen.ZonaID, a.Destino.ZonaID, a.Origen.Direccion, a.Destino.Direccion,
			a.OrigenCoord.Lat, a.OrigenCoord.Lng, a.DestinoCoord.Lat, a.DestinoCoord.Lng, a.Ayudantes,
			a.Origen.Pisos, a.Origen.AscensorUtilizable, a.Origen.DistanciaVehiculoM,
			a.Destino.Pisos, a.Destino.AscensorUtilizable, a.Destino.DistanciaVehiculoM,
			a.Fecha.Format(time.DateOnly), a.FranjaInicio, a.FranjaFin,
		).Scan(&nuevoID)
		if err != nil {
			return err
		}
		for _, o := range a.Objetos {
			if err := insertarObjeto(ctx, tx, nuevoID, o); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("crear solicitud: %w", err)
	}
	return nuevoID, nil
}

func insertarObjeto(ctx context.Context, tx pgx.Tx, solicitudID string, o ObjetoRequest) error {
	var err error
	if o.EsDelCatalogo() {
		// Peso, medidas y restricciones los copia la base del catálogo (0014); los
		// valores de acá son de relleno para los NOT NULL.
		_, err = tx.Exec(ctx, `
			INSERT INTO solicitud_objeto (solicitud_id, objeto_id, cantidad, peso_unitario_kg, largo_m, ancho_m, alto_m)
			VALUES ($1, $2::uuid, $3, 0, 1, 1, 1)`, solicitudID, *o.ObjetoID, o.Cantidad)
	} else {
		verdad := func(b *bool) bool { return b == nil || *b }
		_, err = tx.Exec(ctx, `
			INSERT INTO solicitud_objeto (solicitud_id, nombre_personalizado, cantidad, peso_unitario_kg,
			    largo_m, ancho_m, alto_m, rotacion_horizontal, rotacion_vertical, apilable)
			VALUES ($1, $2, $3, $4::numeric, $5::numeric, $6::numeric, $7::numeric, $8, $9, $10)`,
			solicitudID, *o.NombrePersonalizado, o.Cantidad, *o.PesoUnitarioKg, *o.LargoM, *o.AnchoM, *o.AltoM,
			verdad(o.RotacionHorizontal), verdad(o.RotacionVertical), verdad(o.Apilable))
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == "solicitud_objeto_objeto_id_fkey" {
		return errObjetoInexistente
	}
	return err
}

// detalle lee una solicitud visible para la identidad (RLS). Devuelve
// errNoEncontrada.
func (r *repository) detalle(ctx context.Context, id database.Identity, solicitudID string, hoy time.Time) (SolicitudResponse, error) {
	var s SolicitudResponse
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			SELECT s.id::text, `+estadoSQL+`, to_char(s.fecha_servicio_deseada, 'YYYY-MM-DD'),
			       to_char(s.franja_horaria_inicio, 'HH24:MI'), to_char(s.franja_horaria_fin, 'HH24:MI'),
			       s.origen_zona_id::text, zo.nombre, s.origen_direccion, s.pisos_origen,
			       s.ascensor_utilizable_origen, s.distancia_vehiculo_origen_m::text,
			       s.destino_zona_id::text, zd.nombre, s.destino_direccion, s.pisos_destino,
			       s.ascensor_utilizable_destino, s.distancia_vehiculo_destino_m::text,
			       s.cantidad_ayudantes_solicitados, s.creado_en
			FROM solicitud s
			JOIN zona zo ON zo.id = s.origen_zona_id
			JOIN zona zd ON zd.id = s.destino_zona_id
			WHERE s.id = $2`, hoy.Format(time.DateOnly), solicitudID,
		).Scan(&s.ID, &s.Estado, &s.FechaServicioDeseada, &s.FranjaHorariaInicio, &s.FranjaHorariaFin,
			&s.Origen.ZonaID, &s.Origen.ZonaNombre, &s.Origen.Direccion, &s.Origen.Pisos,
			&s.Origen.AscensorUtilizable, &s.Origen.DistanciaVehiculoM,
			&s.Destino.ZonaID, &s.Destino.ZonaNombre, &s.Destino.Direccion, &s.Destino.Pisos,
			&s.Destino.AscensorUtilizable, &s.Destino.DistanciaVehiculoM,
			&s.CantidadAyudantesSolicitados, &s.CreadoEn)
		if errors.Is(err, pgx.ErrNoRows) {
			return errNoEncontrada
		}
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			SELECT so.id::text, so.objeto_id::text, COALESCE(o.nombre, so.nombre_personalizado), so.cantidad,
			       so.peso_unitario_kg::text, so.largo_m::text, so.ancho_m::text, so.alto_m::text,
			       so.rotacion_horizontal, so.rotacion_vertical, so.apilable
			FROM solicitud_objeto so
			LEFT JOIN objeto o ON o.id = so.objeto_id
			WHERE so.solicitud_id = $1
			ORDER BY COALESCE(o.nombre, so.nombre_personalizado), so.id`, solicitudID)
		if err != nil {
			return err
		}
		s.Objetos, err = pgx.CollectRows(rows, pgx.RowToStructByPos[ObjetoResponse])
		return err
	})
	if err != nil {
		return SolicitudResponse{}, fmt.Errorf("leer solicitud: %w", err)
	}
	return s, nil
}

// propias lista las solicitudes del Cliente, las más nuevas primero.
func (r *repository) propias(ctx context.Context, id database.Identity, hoy time.Time) ([]SolicitudResumen, error) {
	out := []SolicitudResumen{}
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT s.id::text, `+estadoSQL+`, to_char(s.fecha_servicio_deseada, 'YYYY-MM-DD'),
			       to_char(s.franja_horaria_inicio, 'HH24:MI'), to_char(s.franja_horaria_fin, 'HH24:MI'),
			       zo.nombre, zd.nombre,
			       (SELECT COALESCE(sum(cantidad), 0)::int FROM solicitud_objeto WHERE solicitud_id = s.id),
			       s.creado_en
			FROM solicitud s
			JOIN zona zo ON zo.id = s.origen_zona_id
			JOIN zona zd ON zd.id = s.destino_zona_id
			WHERE s.cliente_id = (select auth.uid())
			ORDER BY s.creado_en DESC`, hoy.Format(time.DateOnly))
		if err != nil {
			return err
		}
		lista, err := pgx.CollectRows(rows, pgx.RowToStructByPos[SolicitudResumen])
		out = append(out, lista...)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listar solicitudes: %w", err)
	}
	return out, nil
}

// cancelar pasa a cancelada una solicitud publicada del Cliente y sus ofertas
// pendientes a no_seleccionada (D-20). Devuelve errNoEncontrada o
// errNoCancelable.
func (r *repository) cancelar(ctx context.Context, id database.Identity, solicitudID string) error {
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		var estado string
		err := tx.QueryRow(ctx, `SELECT estado_codigo FROM solicitud
			WHERE id = $1 AND cliente_id = (select auth.uid()) FOR UPDATE`, solicitudID).Scan(&estado)
		if errors.Is(err, pgx.ErrNoRows) {
			return errNoEncontrada
		}
		if err != nil {
			return err
		}
		if estado != "publicada" {
			return errNoCancelable
		}
		if _, err := tx.Exec(ctx, `UPDATE solicitud SET estado_codigo = 'cancelada' WHERE id = $1`, solicitudID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE oferta SET estado_codigo = 'no_seleccionada'
			WHERE solicitud_id = $1 AND estado_codigo = 'pendiente'`, solicitudID)
		return err
	})
	if err != nil {
		return fmt.Errorf("cancelar solicitud: %w", err)
	}
	return nil
}

// republicar crea una solicitud nueva copiando una vencida del Cliente (datos y
// objetos) con otra fecha y franja. La vencida queda como registro (D-20).
// Devuelve el id nuevo, errNoEncontrada o errNoVencida.
func (r *repository) republicar(ctx context.Context, id database.Identity, solicitudID string, hoy, fecha time.Time, inicio, fin *string) (string, error) {
	var nuevoID string
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		var vencida bool
		err := tx.QueryRow(ctx, `SELECT estado_codigo = 'publicada' AND fecha_servicio_deseada < $2::date
			FROM solicitud WHERE id = $1 AND cliente_id = (select auth.uid())`,
			solicitudID, hoy.Format(time.DateOnly)).Scan(&vencida)
		if errors.Is(err, pgx.ErrNoRows) {
			return errNoEncontrada
		}
		if err != nil {
			return err
		}
		if !vencida {
			return errNoVencida
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO solicitud (cliente_id, origen_zona_id, destino_zona_id, origen_direccion, destino_direccion,
			    origen_lat, origen_lng, destino_lat, destino_lng, cantidad_ayudantes_solicitados,
			    pisos_origen, ascensor_utilizable_origen, distancia_vehiculo_origen_m,
			    pisos_destino, ascensor_utilizable_destino, distancia_vehiculo_destino_m,
			    fecha_servicio_deseada, franja_horaria_inicio, franja_horaria_fin)
			SELECT cliente_id, origen_zona_id, destino_zona_id, origen_direccion, destino_direccion,
			       origen_lat, origen_lng, destino_lat, destino_lng, cantidad_ayudantes_solicitados,
			       pisos_origen, ascensor_utilizable_origen, distancia_vehiculo_origen_m,
			       pisos_destino, ascensor_utilizable_destino, distancia_vehiculo_destino_m,
			       $2::date, $3::time, $4::time
			FROM solicitud WHERE id = $1
			RETURNING id::text`, solicitudID, fecha.Format(time.DateOnly), inicio, fin).Scan(&nuevoID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO solicitud_objeto (solicitud_id, objeto_id, nombre_personalizado, cantidad, peso_unitario_kg,
			    largo_m, ancho_m, alto_m, rotacion_horizontal, rotacion_vertical, apilable)
			SELECT $2, objeto_id, nombre_personalizado, cantidad, peso_unitario_kg,
			       largo_m, ancho_m, alto_m, rotacion_horizontal, rotacion_vertical, apilable
			FROM solicitud_objeto WHERE solicitud_id = $1`, solicitudID, nuevoID)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("republicar solicitud: %w", err)
	}
	return nuevoID, nil
}
