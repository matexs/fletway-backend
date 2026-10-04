package vehiculo

import (
	"context"
	"errors"
	"fmt"

	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

var (
	errNoEsTransportista = httpx.Forbidden("no_es_transportista",
		"sólo un Transportista registrado puede administrar vehículos")
	errNoEncontrado = httpx.NotFound("vehiculo_no_encontrado", "no existe el vehículo o no es tuyo")
	errDuplicada    = httpx.Conflict("patente_duplicada", "ya hay un vehículo registrado con esa patente")
	errTipo         = httpx.BadRequest("tipo_vehiculo_invalido", "el tipo de vehículo no existe")
)

// Service administra los vehículos del Transportista (RF-18).
type Service struct {
	repo *repository
}

// NewService crea el Service sobre la base dada.
func NewService(db *database.DB) *Service {
	return &Service{repo: &repository{db: db}}
}

// Tipos lista los tipos de vehículo con sus medidas estándar (D-32).
func (s *Service) Tipos(ctx context.Context, id database.Identity) ([]TipoVehiculoResponse, error) {
	return s.repo.tipos(ctx, id)
}

// Crear registra un vehículo del Transportista, activo. Devuelve no_es_transportista, tipo_vehiculo_invalido o
// patente_duplicada. Escribe en la base.
func (s *Service) Crear(ctx context.Context, id database.Identity, in CrearVehiculoRequest) (VehiculoResponse, error) {
	if err := s.exigirTransportista(ctx, id); err != nil {
		return VehiculoResponse{}, err
	}
	v, err := s.repo.crear(ctx, id, in)
	switch {
	case errors.Is(err, errPatenteDuplicada):
		return VehiculoResponse{}, errDuplicada
	case errors.Is(err, errTipoInexistente):
		return VehiculoResponse{}, errTipo
	case err != nil:
		return VehiculoResponse{}, fmt.Errorf("crear vehículo: %w", err)
	}
	return v, nil
}

// Propios lista los vehículos del Transportista. Devuelve no_es_transportista.
func (s *Service) Propios(ctx context.Context, id database.Identity) ([]VehiculoResponse, error) {
	if err := s.exigirTransportista(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.propios(ctx, id)
}

// CambiarActivo activa o desactiva un vehículo propio; uno inactivo no cuenta
// para el matchmaking ni para ofertar (D-21). Devuelve vehiculo_no_encontrado.
func (s *Service) CambiarActivo(ctx context.Context, id database.Identity, vehiculoID string, activo bool) (VehiculoResponse, error) {
	v, err := s.repo.cambiarActivo(ctx, id, vehiculoID, activo)
	if errors.Is(err, errVehiculoNoEncontrado) {
		return VehiculoResponse{}, errNoEncontrado
	}
	return v, err
}

func (s *Service) exigirTransportista(ctx context.Context, id database.Identity) error {
	es, err := s.repo.esTransportista(ctx, id)
	if err != nil {
		return err
	}
	if !es {
		return errNoEsTransportista
	}
	return nil
}
