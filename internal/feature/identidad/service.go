package identidad

import (
	"context"
	"errors"
	"fmt"

	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

var (
	errUsuarioNoEncontrado = httpx.NotFound("usuario_no_encontrado",
		"la cuenta no tiene perfil de usuario")
	errRolNoCorresponde = httpx.Conflict("rol_no_corresponde",
		"la cuenta no tiene el rol de este registro")
	errCuentaInactiva    = httpx.Forbidden("cuenta_inactiva", "la cuenta está desactivada")
	errNoEsTransportista = httpx.Forbidden("no_es_transportista",
		"sólo un Transportista registrado tiene disponibilidad")
)

// Service implementa el alta de cuenta y el perfil del usuario autenticado (D-18).
type Service struct {
	repo *repository
}

// NewService crea el Service sobre la base dada.
func NewService(db *database.DB) *Service {
	return &Service{repo: &repository{db: db}}
}

// Me devuelve el perfil del usuario autenticado. Devuelve un *httpx.APIError
// usuario_no_encontrado si el usuario de Auth no tiene fila en usuario.
func (s *Service) Me(ctx context.Context, id database.Identity) (Perfil, error) {
	p, err := s.repo.perfil(ctx, id)
	if errors.Is(err, errSinUsuario) {
		return Perfil{}, errUsuarioNoEncontrado
	}
	if err != nil {
		return Perfil{}, fmt.Errorf("obtener perfil: %w", err)
	}
	return p, nil
}

// Registrar completa el alta de la cuenta creando su fila de rol (RF-05 para rol
// cliente, RF-16 para rol transportista, que queda pendiente de habilitación). El rol
// tiene que coincidir con usuario.rol, que fijó el trigger de alta: si no, devuelve
// rol_no_corresponde. También devuelve usuario_no_encontrado o cuenta_inactiva.
// Es idempotente: creada es false si la fila ya existía. Escribe en la base.
func (s *Service) Registrar(ctx context.Context, id database.Identity, rol string) (p Perfil, creada bool, err error) {
	p, err = s.Me(ctx, id)
	if err != nil {
		return Perfil{}, false, err
	}
	if !p.Activo {
		return Perfil{}, false, errCuentaInactiva
	}
	if p.Rol != rol {
		return Perfil{}, false, errRolNoCorresponde
	}
	creada, err = s.repo.crearFilaDeRol(ctx, id, rol)
	if err != nil {
		return Perfil{}, false, fmt.Errorf("registrar %s: %w", rol, err)
	}
	p, err = s.Me(ctx, id)
	if err != nil {
		return Perfil{}, false, err
	}
	return p, creada, nil
}

// CambiarDisponibilidad prende o apaga el interruptor "estoy tomando trabajos"
// del Transportista (D-21) y devuelve el perfil actualizado. Devuelve
// no_es_transportista. Escribe en la base.
func (s *Service) CambiarDisponibilidad(ctx context.Context, id database.Identity, disponible bool) (Perfil, error) {
	err := s.repo.cambiarDisponible(ctx, id, disponible)
	if errors.Is(err, errSinTransportista) {
		return Perfil{}, errNoEsTransportista
	}
	if err != nil {
		return Perfil{}, err
	}
	return s.Me(ctx, id)
}
