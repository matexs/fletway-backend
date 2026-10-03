package oferta

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/ruteo"
)

var (
	errNoEncontrada         = errors.New("oferta: no encontrada")
	errNoRetirable          = errors.New("oferta: no está pendiente")
	errDuplicada            = errors.New("oferta: ya hay una vigente con ese vehículo")
	errVehiculoNoEncontrado = errors.New("oferta: vehículo no encontrado")
	errVehiculoInactivo     = errors.New("oferta: vehículo inactivo")
	errSinCostos            = errors.New("oferta: vehículo sin costos cargados")
	errSolicitudInexistente = errors.New("oferta: solicitud no visible")
)

type repository struct {
	db *database.DB
}

// perfil es lo que se necesita saber del Transportista para dejarlo ofertar.
type perfil struct {
	Existe, Habilitado, Disponible bool
}

func (r *repository) perfil(ctx context.Context, id database.Identity) (perfil, error) {
	var p perfil
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			SELECT true, estado_habilitacion_codigo = 'habilitado', disponible
			FROM transportista WHERE usuario_id = (select auth.uid())`).Scan(&p.Existe, &p.Habilitado, &p.Disponible)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	})
	if err != nil {
		return perfil{}, fmt.Errorf("leer transportista: %w", err)
	}
	return p, nil
}

// compatible indica si la solicitud está entre las compatibles con el
// Transportista (RN-04; la regla vive en fn_solicitudes_compatibles, 0015).
func (r *repository) compatible(ctx context.Context, id database.Identity, solicitudID string) (bool, error) {
	var ok bool
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM fn_solicitudes_compatibles() c WHERE c = $1::uuid)`, solicitudID).Scan(&ok)
	})
	if err != nil {
		return false, fmt.Errorf("verificar compatibilidad: %w", err)
	}
	return ok, nil
}

// entrada son todos los datos del cálculo, leídos en una transacción.
type entrada struct {
	Parametros      Parametros
	Vehiculo        Vehiculo
	Costo           VehiculoCosto
	Origen, Destino ruteo.Coordenada
	AccesoOrigen    Acceso
	AccesoDestino   Acceso
	Carga           []Item
}

// entrada lee la configuración vigente, el vehículo con sus costos y la
// solicitud con sus objetos. Devuelve errVehiculoNoEncontrado,
// errVehiculoInactivo, errSinCostos o errSolicitudInexistente.
func (r *repository) entrada(ctx context.Context, id database.Identity, solicitudID, vehiculoID string) (entrada, error) {
	var e entrada
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		if err := leerParametros(ctx, tx, &e.Parametros); err != nil {
			return err
		}
		if err := leerVehiculo(ctx, tx, vehiculoID, &e); err != nil {
			return err
		}
		err := tx.QueryRow(ctx, `
			SELECT origen_lat::float8, origen_lng::float8, destino_lat::float8, destino_lng::float8,
			       pisos_origen, ascensor_utilizable_origen, distancia_vehiculo_origen_m::float8,
			       pisos_destino, ascensor_utilizable_destino, distancia_vehiculo_destino_m::float8
			FROM solicitud WHERE id = $1`, solicitudID,
		).Scan(&e.Origen.Lat, &e.Origen.Lng, &e.Destino.Lat, &e.Destino.Lng,
			&e.AccesoOrigen.Pisos, &e.AccesoOrigen.AscensorUtilizable, &e.AccesoOrigen.DistanciaVehiculoM,
			&e.AccesoDestino.Pisos, &e.AccesoDestino.AscensorUtilizable, &e.AccesoDestino.DistanciaVehiculoM)
		if errors.Is(err, pgx.ErrNoRows) {
			return errSolicitudInexistente
		}
		if err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			SELECT COALESCE(o.nombre, so.nombre_personalizado), so.peso_unitario_kg::float8,
			       so.largo_m::float8, so.ancho_m::float8, so.alto_m::float8,
			       so.rotacion_horizontal, so.rotacion_vertical, so.apilable, so.cantidad
			FROM solicitud_objeto so
			LEFT JOIN objeto o ON o.id = so.objeto_id
			WHERE so.solicitud_id = $1
			ORDER BY so.id`, solicitudID)
		if err != nil {
			return err
		}
		e.Carga, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (Item, error) {
			var it Item
			o := &it.Objeto
			err := row.Scan(&o.Nombre, &o.PesoKg, &o.LargoM, &o.AnchoM, &o.AltoM,
				&o.RotacionHorizontal, &o.RotacionVertical, &o.Apilable, &it.Cantidad)
			return it, err
		})
		return err
	})
	if err != nil {
		return entrada{}, fmt.Errorf("leer datos de la oferta: %w", err)
	}
	return e, nil
}

// leerParametros lee las filas vigentes de config_*; si falta alguna es un error
// de configuración de la plataforma, no del usuario.
func leerParametros(ctx context.Context, tx pgx.Tx, p *Parametros) error {
	l := &p.Laboral
	err := tx.QueryRow(ctx, `
		SELECT salario_basico_chofer::float8, salario_basico_ayudante::float8,
		       adicionales_pct_chofer::float8, adicionales_pct_ayudante::float8, viaticos_diarios::float8,
		       contribuciones_seg_social_pct::float8, obra_social_pct::float8, art_pct::float8,
		       seguro_vida_mensual::float8, horas_mensuales::float8, horas_diarias::float8
		FROM config_costo_laboral WHERE vigente_hasta IS NULL`,
	).Scan(&l.SalarioBasicoChofer, &l.SalarioBasicoAyudante, &l.AdicionalesPctChofer, &l.AdicionalesPctAyudante,
		&l.ViaticosDiarios, &l.ContribucionesSegSocialPct, &l.ObraSocialPct, &l.ArtPct,
		&l.SeguroVidaMensual, &l.HorasMensuales, &l.HorasDiarias)
	if err != nil {
		return fmt.Errorf("config_costo_laboral vigente: %w", err)
	}
	o := &p.Operacion
	err = tx.QueryRow(ctx, `
		SELECT tiempo_base_operacion_min::float8, tiempo_espera_min::float8, tiempo_por_objeto_min::float8,
		       tiempo_por_kg_min::float8, tiempo_por_m3_min::float8, tiempo_por_metro_min::float8,
		       tiempo_por_escalera_min::float8, eficiencia_ayudante::float8
		FROM config_operacion WHERE vigente_hasta IS NULL`,
	).Scan(&o.TiempoBaseOperacionMin, &o.TiempoEsperaMin, &o.TiempoPorObjetoMin, &o.TiempoPorKgMin,
		&o.TiempoPorM3Min, &o.TiempoPorMetroMin, &o.TiempoPorEscaleraMin, &o.EficienciaAyudante)
	if err != nil {
		return fmt.Errorf("config_operacion vigente: %w", err)
	}
	err = tx.QueryRow(ctx, `
		SELECT (SELECT margen_pct::float8 FROM config_margen WHERE vigente_hasta IS NULL),
		       (SELECT porcentaje::float8 FROM config_comision WHERE vigente_hasta IS NULL),
		       (SELECT iva_pct::float8 FROM config_impuesto WHERE vigente_hasta IS NULL)`,
	).Scan(&p.MargenPct, &p.ComisionPct, &p.IvaPct)
	if err != nil {
		return fmt.Errorf("margen, comisión e IVA vigentes: %w", err)
	}
	return nil
}

func leerVehiculo(ctx context.Context, tx pgx.Tx, vehiculoID string, e *entrada) error {
	var activo, conCostos bool
	v := &e.Vehiculo
	c := &e.Costo
	err := tx.QueryRow(ctx, `
		SELECT v.patente, v.peso_maximo_kg::float8, v.largo_util_m::float8, v.ancho_util_m::float8,
		       v.alto_util_m::float8, v.activo, c.vehiculo_id IS NOT NULL,
		       COALESCE(c.combustible_precio_l, 0)::float8, COALESCE(c.rendimiento_km_l, 1)::float8,
		       COALESCE(c.cantidad_neumaticos, 0), COALESCE(c.costo_neumatico, 0)::float8,
		       COALESCE(c.vida_neumatico_km, 1)::float8, COALESCE(c.costo_mantenimiento_km, 0)::float8,
		       COALESCE(c.valor_compra, 0)::float8, COALESCE(c.valor_residual, 0)::float8,
		       COALESCE(c.vida_util_km, 1)::float8, COALESCE(c.seguro_mensual, 0)::float8,
		       COALESCE(c.patente_mensual, 0)::float8
		FROM vehiculo v
		LEFT JOIN vehiculo_costo c ON c.vehiculo_id = v.id
		WHERE v.id = $1 AND v.transportista_id = (select auth.uid())`, vehiculoID,
	).Scan(&v.Patente, &v.PesoUtilKg, &v.LargoUtilM, &v.AnchoUtilM, &v.AltoUtilM, &activo, &conCostos,
		&c.CombustiblePrecioL, &c.RendimientoKmL, &c.CantidadNeumaticos, &c.CostoNeumatico,
		&c.VidaNeumaticoKm, &c.CostoMantenimientoKm, &c.ValorCompra, &c.ValorResidual,
		&c.VidaUtilKm, &c.SeguroMensual, &c.PatenteMensual)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return errVehiculoNoEncontrado
	case err != nil:
		return err
	case !activo:
		return errVehiculoInactivo
	case !conCostos:
		// Nunca se calcula con costos en cero (ALGORITMO_COTIZACION.md §7).
		return errSinCostos
	}
	return nil
}

// alta es la oferta calculada, lista para guardar.
type alta struct {
	SolicitudID, VehiculoID string
	Ayudantes               int
	Viajes                  int
	Desglose                Desglose
}

// crear inserta la oferta y su desglose en una transacción: la base rechaza una
// oferta sin desglose (0016). Devuelve el id o errDuplicada.
func (r *repository) crear(ctx context.Context, id database.Identity, a alta) (string, error) {
	var nuevoID string
	d := a.Desglose
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO oferta (solicitud_id, transportista_id, vehiculo_id, cantidad_viajes, cantidad_ayudantes,
			    precio_calculado)
			VALUES ($1, (select auth.uid()), $2, $3, $4, $5::numeric)
			RETURNING id::text`,
			a.SolicitudID, a.VehiculoID, a.Viajes, a.Ayudantes, dec(d.PrecioFinal)).Scan(&nuevoID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO oferta_costo (oferta_id, distancia_km, duracion_ruta_h, duracion_operacion_h,
			    costo_laboral, costo_vehiculo, costos_adicionales, costo_operativo, margen_pct, precio_neto,
			    porcentaje_comision, iva_pct)
			VALUES ($1, $2::numeric, $3::numeric, $4::numeric, $5::numeric, $6::numeric, $7::numeric,
			        $8::numeric, $9::numeric, $10::numeric, $11::numeric, $12::numeric)`,
			nuevoID, dec(d.DistanciaKm), dec(d.DuracionRutaH), dec(d.DuracionOperacionH),
			dec(d.CostoLaboral), dec(d.CostoVehiculo), dec(d.CostosAdicionales), dec(d.CostoOperativo),
			dec(d.MargenPct), dec(d.PrecioNeto), dec(d.ComisionPct), dec(d.IvaPct))
		return err
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == "uq_oferta_vigente_por_vehiculo" {
		return "", errDuplicada
	}
	if err != nil {
		return "", fmt.Errorf("crear oferta: %w", err)
	}
	return nuevoID, nil
}

// retirar pasa a retirada una oferta pendiente del Transportista. Devuelve
// errNoEncontrada o errNoRetirable.
func (r *repository) retirar(ctx context.Context, id database.Identity, ofertaID string) error {
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		var estado string
		err := tx.QueryRow(ctx, `SELECT estado_codigo FROM oferta
			WHERE id = $1 AND transportista_id = (select auth.uid()) FOR UPDATE`, ofertaID).Scan(&estado)
		if errors.Is(err, pgx.ErrNoRows) {
			return errNoEncontrada
		}
		if err != nil {
			return err
		}
		if estado != "pendiente" {
			return errNoRetirable
		}
		_, err = tx.Exec(ctx, `UPDATE oferta SET estado_codigo = 'retirada' WHERE id = $1`, ofertaID)
		return err
	})
	if err != nil {
		return fmt.Errorf("retirar oferta: %w", err)
	}
	return nil
}

// selectOferta lee las ofertas del Transportista con su desglose. $1 es la fecha
// de hoy en Argentina, para calcular si la solicitud venció (D-20).
const selectOferta = `
	SELECT o.id::text, o.estado_codigo, s.id::text,
	       CASE WHEN s.estado_codigo = 'publicada' AND s.fecha_servicio_deseada < $1::date
	            THEN 'vencida' ELSE s.estado_codigo END,
	       to_char(s.fecha_servicio_deseada, 'YYYY-MM-DD'), zo.nombre, zd.nombre,
	       v.id::text, v.patente, tv.nombre, o.cantidad_viajes, o.cantidad_ayudantes, o.precio_calculado::text,
	       c.distancia_km::text, c.duracion_ruta_h::text, c.duracion_operacion_h::text, c.costo_laboral::text,
	       c.costo_vehiculo::text, c.costos_adicionales::text, c.costo_operativo::text, c.margen_pct::text,
	       c.precio_neto::text, c.porcentaje_comision::text, c.iva_pct::text, o.creado_en
	FROM oferta o
	JOIN oferta_costo c ON c.oferta_id = o.id
	JOIN solicitud s ON s.id = o.solicitud_id
	JOIN zona zo ON zo.id = s.origen_zona_id
	JOIN zona zd ON zd.id = s.destino_zona_id
	JOIN vehiculo v ON v.id = o.vehiculo_id
	JOIN tipo_vehiculo tv ON tv.id = v.tipo_vehiculo_id
	WHERE o.transportista_id = (select auth.uid())`

func escanearOferta(row pgx.CollectableRow) (OfertaResponse, error) {
	var o OfertaResponse
	d := &o.Desglose
	err := row.Scan(&o.ID, &o.Estado, &o.SolicitudID, &o.SolicitudEstado, &o.FechaServicio,
		&o.OrigenZona, &o.DestinoZona, &o.VehiculoID, &o.VehiculoPatente, &o.VehiculoTipo,
		&o.CantidadViajes, &o.CantidadAyudantes, &o.PrecioCalculado,
		&d.DistanciaKm, &d.DuracionRutaH, &d.DuracionOperacionH, &d.CostoLaboral, &d.CostoVehiculo,
		&d.CostosAdicionales, &d.CostoOperativo, &d.MargenPct, &d.PrecioNeto, &d.PorcentajeComision,
		&d.IvaPct, &o.CreadoEn)
	return o, err
}

// propias lista las ofertas del Transportista, las más nuevas primero.
func (r *repository) propias(ctx context.Context, id database.Identity, hoy time.Time) ([]OfertaResponse, error) {
	out := []OfertaResponse{}
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, selectOferta+` ORDER BY o.creado_en DESC`, hoy.Format(time.DateOnly))
		if err != nil {
			return err
		}
		lista, err := pgx.CollectRows(rows, escanearOferta)
		out = append(out, lista...)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listar ofertas: %w", err)
	}
	return out, nil
}

// detalle lee una oferta del Transportista. Devuelve errNoEncontrada.
func (r *repository) detalle(ctx context.Context, id database.Identity, ofertaID string, hoy time.Time) (OfertaResponse, error) {
	var o OfertaResponse
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, selectOferta+` AND o.id = $2`, hoy.Format(time.DateOnly), ofertaID)
		if err != nil {
			return err
		}
		o, err = pgx.CollectExactlyOneRow(rows, escanearOferta)
		if errors.Is(err, pgx.ErrNoRows) {
			return errNoEncontrada
		}
		return err
	})
	if err != nil {
		return OfertaResponse{}, fmt.Errorf("leer oferta: %w", err)
	}
	return o, nil
}
