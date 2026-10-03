package identidad

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/matexs/fletway-backend/internal/platform/database"
)

// Roles posibles de usuario.rol.
const (
	RolCliente       = "cliente"
	RolTransportista = "transportista"
	RolAdministrador = "administrador"
)

// errSinTransportista indica que la cuenta no tiene fila transportista.
var errSinTransportista = errors.New("identidad: la cuenta no es de Transportista")

// errSinUsuario indica que el usuario de Auth no tiene fila en usuario (no pasó por el
// trigger de alta, D-18).
var errSinUsuario = errors.New("identidad: usuario sin fila en la tabla usuario")

// Perfil es la fila de usuario del usuario autenticado más el estado de su fila de rol.
type Perfil struct {
	UsuarioID      string
	Email          string
	NombreCompleto string
	Telefono       string
	Rol            string
	Activo         bool
	// RegistroCompleto indica si existe la fila de la tabla de su rol.
	RegistroCompleto bool
	// EstadoHabilitacion sólo tiene valor para un Transportista registrado.
	EstadoHabilitacion *string
	// Disponible sólo tiene valor para un Transportista registrado.
	Disponible *bool
}

type repository struct {
	db *database.DB
}

// perfil lee el perfil del usuario de la identidad. Devuelve errSinUsuario si no existe.
func (r *repository) perfil(ctx context.Context, id database.Identity) (Perfil, error) {
	var p Perfil
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT u.id::text, u.email::text, u.nombre_completo, u.telefono, u.rol, u.activo,
			       CASE u.rol
			         WHEN 'cliente' THEN c.usuario_id IS NOT NULL
			         WHEN 'transportista' THEN t.usuario_id IS NOT NULL
			         ELSE a.usuario_id IS NOT NULL
			       END,
			       CASE WHEN u.rol = 'transportista' THEN t.estado_habilitacion_codigo END,
			       CASE WHEN u.rol = 'transportista' THEN t.disponible END
			FROM usuario u
			LEFT JOIN cliente c ON c.usuario_id = u.id
			LEFT JOIN transportista t ON t.usuario_id = u.id
			LEFT JOIN administrador a ON a.usuario_id = u.id
			WHERE u.id = (select auth.uid())`,
		).Scan(&p.UsuarioID, &p.Email, &p.NombreCompleto, &p.Telefono, &p.Rol, &p.Activo,
			&p.RegistroCompleto, &p.EstadoHabilitacion, &p.Disponible)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Perfil{}, errSinUsuario
	}
	if err != nil {
		return Perfil{}, fmt.Errorf("leer perfil: %w", err)
	}
	return p, nil
}

// crearFilaDeRol inserta la fila de cliente o transportista del usuario. Es idempotente:
// devuelve creada=false si ya existía. El estado inicial del Transportista lo fija la
// base (pendiente, trg_proteger_alta_transportista).
func (r *repository) crearFilaDeRol(ctx context.Context, id database.Identity, rol string) (bool, error) {
	var sql string
	switch rol {
	case RolCliente:
		sql = `INSERT INTO cliente (usuario_id) VALUES ((select auth.uid())) ON CONFLICT DO NOTHING`
	case RolTransportista:
		sql = `INSERT INTO transportista (usuario_id) VALUES ((select auth.uid())) ON CONFLICT DO NOTHING`
	default:
		return false, fmt.Errorf("crear fila de rol: rol %q no se registra por la API", rol)
	}
	var creada bool
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, sql)
		creada = tag.RowsAffected() == 1
		return err
	})
	if err != nil {
		return false, fmt.Errorf("crear fila de %s: %w", rol, err)
	}
	return creada, nil
}

// cambiarDisponible actualiza transportista.disponible del usuario. Devuelve
// errSinTransportista si no tiene fila transportista.
func (r *repository) cambiarDisponible(ctx context.Context, id database.Identity, disponible bool) error {
	err := r.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE transportista SET disponible = $1
			WHERE usuario_id = (select auth.uid())`, disponible)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errSinTransportista
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("cambiar disponibilidad: %w", err)
	}
	return nil
}
