package vehiculo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/matexs/fletway-backend/internal/platform/database"
)

var (
	errVehiculoNoEncontrado = errors.New("vehiculo: no encontrado")
	errPatenteDuplicada     = errors.New("vehiculo: patente duplicada")
	errTipoInexistente      = errors.New("vehiculo: tipo inexistente")
	errSinCostos            = errors.New("vehiculo: sin costos cargados")
)

type repository struct {
	db *database.DB
}

// esTransportista indica si la identidad tiene fila transportista.
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

func (r *repository) tipos(ctx context.Context, id database.Identity) ([]TipoVehiculoResponse, error) {
	var out []TipoVehiculoResponse
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id::text, nombre, largo_estandar_m::text, ancho_estandar_m::text, alto_estandar_m::text,
			       peso_maximo_estandar_kg::text
			FROM tipo_vehiculo ORDER BY volumen_estandar_m3`)
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (TipoVehiculoResponse, error) {
			var t TipoVehiculoResponse
			err := row.Scan(&t.ID, &t.Nombre, &t.LargoEstandarM, &t.AnchoEstandarM, &t.AltoEstandarM,
				&t.PesoMaximoEstandarKg)
			return t, err
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listar tipos de vehículo: %w", err)
	}
	return out, nil
}

const selectVehiculo = `
	SELECT v.id::text, v.tipo_vehiculo_id::text, tv.nombre, v.patente, v.marca, v.modelo,
	       v.largo_util_m::text, v.ancho_util_m::text, v.alto_util_m::text, v.peso_maximo_kg::text,
	       v.activo, EXISTS (SELECT 1 FROM vehiculo_costo c WHERE c.vehiculo_id = v.id), v.creado_en
	FROM vehiculo v
	JOIN tipo_vehiculo tv ON tv.id = v.tipo_vehiculo_id`

func escanearVehiculo(row pgx.Row) (VehiculoResponse, error) {
	var v VehiculoResponse
	err := row.Scan(&v.ID, &v.TipoVehiculoID, &v.TipoVehiculoNombre, &v.Patente, &v.Marca, &v.Modelo,
		&v.LargoUtilM, &v.AnchoUtilM, &v.AltoUtilM, &v.PesoMaximoKg, &v.Activo, &v.TieneCostos, &v.CreadoEn)
	return v, err
}

// crear inserta el vehículo del Transportista de la identidad. Devuelve
// errPatenteDuplicada o errTipoInexistente.
func (r *repository) crear(ctx context.Context, id database.Identity, in CrearVehiculoRequest) (VehiculoResponse, error) {
	var v VehiculoResponse
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		var nuevoID string
		err := tx.QueryRow(ctx, `
			INSERT INTO vehiculo (transportista_id, tipo_vehiculo_id, patente, marca, modelo,
			                      largo_util_m, ancho_util_m, alto_util_m, peso_maximo_kg)
			VALUES ((select auth.uid()), $1::uuid, $2, $3, $4, $5::numeric, $6::numeric, $7::numeric, $8::numeric)
			RETURNING id::text`,
			in.TipoVehiculoID, in.Patente, in.Marca, in.Modelo,
			in.LargoUtilM, in.AnchoUtilM, in.AltoUtilM, in.PesoMaximoKg,
		).Scan(&nuevoID)
		if err != nil {
			return traducir(err)
		}
		v, err = escanearVehiculo(tx.QueryRow(ctx, selectVehiculo+` WHERE v.id = $1`, nuevoID))
		return err
	})
	if err != nil {
		return VehiculoResponse{}, fmt.Errorf("crear vehículo: %w", err)
	}
	return v, nil
}

// traducir convierte las violaciones de constraint esperables en errores del
// paquete.
func traducir(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == "23505" && pgErr.ConstraintName == "vehiculo_patente_key":
			return errPatenteDuplicada
		case pgErr.Code == "23503" && pgErr.ConstraintName == "vehiculo_tipo_vehiculo_id_fkey",
			pgErr.Code == "22P02":
			return errTipoInexistente
		}
	}
	return err
}

// propios lista los vehículos del Transportista, los activos primero.
func (r *repository) propios(ctx context.Context, id database.Identity) ([]VehiculoResponse, error) {
	var out []VehiculoResponse
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, selectVehiculo+`
			WHERE v.transportista_id = (select auth.uid())
			ORDER BY v.activo DESC, v.creado_en`)
		if err != nil {
			return err
		}
		out, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (VehiculoResponse, error) {
			return escanearVehiculo(row)
		})
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("listar vehículos: %w", err)
	}
	return out, nil
}

// cambiarActivo activa o desactiva un vehículo propio. Devuelve
// errVehiculoNoEncontrado si no existe o no es del Transportista.
func (r *repository) cambiarActivo(ctx context.Context, id database.Identity, vehiculoID string, activo bool) (VehiculoResponse, error) {
	var v VehiculoResponse
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE vehiculo SET activo = $2
			WHERE id = $1 AND transportista_id = (select auth.uid())`, vehiculoID, activo)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errVehiculoNoEncontrado
		}
		v, err = escanearVehiculo(tx.QueryRow(ctx, selectVehiculo+` WHERE v.id = $1`, vehiculoID))
		return err
	})
	if err != nil {
		return VehiculoResponse{}, fmt.Errorf("cambiar activo: %w", err)
	}
	return v, nil
}

const columnasCostos = `combustible_precio_l::text, rendimiento_km_l::text, cantidad_neumaticos,
	costo_neumatico::text, vida_neumatico_km::text, costo_mantenimiento_km::text, valor_compra::text,
	valor_residual::text, vida_util_km::text, seguro_mensual::text, patente_mensual::text, actualizado_en`

func escanearCostos(row pgx.Row) (CostosResponse, error) {
	var c CostosResponse
	err := row.Scan(&c.CombustiblePrecioL, &c.RendimientoKmL, &c.CantidadNeumaticos, &c.CostoNeumatico,
		&c.VidaNeumaticoKm, &c.CostoMantenimientoKm, &c.ValorCompra, &c.ValorResidual, &c.VidaUtilKm,
		&c.SeguroMensual, &c.PatenteMensual, &c.ActualizadoEn)
	return c, err
}

// esPropio indica si el vehículo existe y es del Transportista de la identidad.
func esPropio(ctx context.Context, tx pgx.Tx, vehiculoID string) (bool, error) {
	var es bool
	err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM vehiculo
		WHERE id = $1 AND transportista_id = (select auth.uid()))`, vehiculoID).Scan(&es)
	return es, err
}

// guardarCostos crea o reemplaza los costos de un vehículo propio. Devuelve
// errVehiculoNoEncontrado si no existe o no es del Transportista.
func (r *repository) guardarCostos(ctx context.Context, id database.Identity, vehiculoID string, c CostosRequest) (CostosResponse, error) {
	var out CostosResponse
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		ok, err := esPropio(ctx, tx, vehiculoID)
		if err != nil {
			return err
		}
		if !ok {
			return errVehiculoNoEncontrado
		}
		out, err = escanearCostos(tx.QueryRow(ctx, `
			INSERT INTO vehiculo_costo (vehiculo_id, combustible_precio_l, rendimiento_km_l, cantidad_neumaticos,
			    costo_neumatico, vida_neumatico_km, costo_mantenimiento_km, valor_compra, valor_residual,
			    vida_util_km, seguro_mensual, patente_mensual, actualizado_en)
			VALUES ($1, $2::numeric, $3::numeric, $4, $5::numeric, $6::numeric, $7::numeric, $8::numeric,
			        $9::numeric, $10::numeric, $11::numeric, $12::numeric, now())
			ON CONFLICT (vehiculo_id) DO UPDATE SET
			    combustible_precio_l = EXCLUDED.combustible_precio_l,
			    rendimiento_km_l = EXCLUDED.rendimiento_km_l,
			    cantidad_neumaticos = EXCLUDED.cantidad_neumaticos,
			    costo_neumatico = EXCLUDED.costo_neumatico,
			    vida_neumatico_km = EXCLUDED.vida_neumatico_km,
			    costo_mantenimiento_km = EXCLUDED.costo_mantenimiento_km,
			    valor_compra = EXCLUDED.valor_compra,
			    valor_residual = EXCLUDED.valor_residual,
			    vida_util_km = EXCLUDED.vida_util_km,
			    seguro_mensual = EXCLUDED.seguro_mensual,
			    patente_mensual = EXCLUDED.patente_mensual,
			    actualizado_en = now()
			RETURNING `+columnasCostos,
			vehiculoID, c.CombustiblePrecioL, c.RendimientoKmL, c.CantidadNeumaticos,
			c.CostoNeumatico, c.VidaNeumaticoKm, c.CostoMantenimientoKm, c.ValorCompra, c.ValorResidual,
			c.VidaUtilKm, c.SeguroMensual, c.PatenteMensual))
		return err
	})
	if err != nil {
		return CostosResponse{}, fmt.Errorf("guardar costos: %w", err)
	}
	return out, nil
}

// costos lee los costos de un vehículo propio. Devuelve errVehiculoNoEncontrado o
// errSinCostos.
func (r *repository) costos(ctx context.Context, id database.Identity, vehiculoID string) (CostosResponse, error) {
	var out CostosResponse
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		ok, err := esPropio(ctx, tx, vehiculoID)
		if err != nil {
			return err
		}
		if !ok {
			return errVehiculoNoEncontrado
		}
		out, err = escanearCostos(tx.QueryRow(ctx,
			`SELECT `+columnasCostos+` FROM vehiculo_costo WHERE vehiculo_id = $1`, vehiculoID))
		if errors.Is(err, pgx.ErrNoRows) {
			return errSinCostos
		}
		return err
	})
	if err != nil {
		return CostosResponse{}, fmt.Errorf("leer costos: %w", err)
	}
	return out, nil
}
