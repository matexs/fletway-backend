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
	       v.activo, v.creado_en
	FROM vehiculo v
	JOIN tipo_vehiculo tv ON tv.id = v.tipo_vehiculo_id`

func escanearVehiculo(row pgx.Row) (VehiculoResponse, error) {
	var v VehiculoResponse
	err := row.Scan(&v.ID, &v.TipoVehiculoID, &v.TipoVehiculoNombre, &v.Patente, &v.Marca, &v.Modelo,
		&v.LargoUtilM, &v.AnchoUtilM, &v.AltoUtilM, &v.PesoMaximoKg, &v.Activo, &v.CreadoEn)
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
