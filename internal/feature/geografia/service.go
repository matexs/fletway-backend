package geografia

import (
	"context"
	"errors"
	"fmt"

	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/geocodificacion"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

var (
	errNoEsTransportista = httpx.Forbidden("no_es_transportista",
		"sólo un Transportista registrado puede elegir zonas de trabajo")
	errZonaInvalida = httpx.BadRequest("zona_invalida", "alguna de las zonas no existe")
)

// Service lee el catálogo de zonas, guarda las del Transportista (RN-04) y
// sugiere direcciones (D-35).
type Service struct {
	repo *repository
	geo  geocodificacion.Geocodificador
}

// NewService crea el Service sobre la base dada; geo sugiere direcciones.
func NewService(db *database.DB, geo geocodificacion.Geocodificador) *Service {
	return &Service{repo: &repository{db: db}, geo: geo}
}

// Sugerencias propone direcciones de la zona que empiezan como texto, para el
// autocompletado al publicar (D-35). Devuelve zona_invalida.
func (s *Service) Sugerencias(ctx context.Context, id database.Identity, zonaID, texto string) ([]geocodificacion.Sugerencia, error) {
	z, err := s.repo.zona(ctx, id, zonaID)
	if errors.Is(err, errZonaInexistente) {
		return nil, errZonaInvalida
	}
	if err != nil {
		return nil, err
	}
	out, err := s.geo.Sugerir(ctx, texto, z.nombre, z.provincia)
	if err != nil {
		return nil, fmt.Errorf("sugerir direcciones: %w", err)
	}
	return out, nil
}

// Zonas lista el catálogo, ordenado por provincia y nombre.
func (s *Service) Zonas(ctx context.Context, id database.Identity) ([]ZonaResponse, error) {
	return s.repo.zonas(ctx, id)
}

// DeTransportista devuelve las zonas de trabajo del Transportista. Devuelve
// no_es_transportista.
func (s *Service) DeTransportista(ctx context.Context, id database.Identity) (ZonasTransportista, error) {
	ids, err := s.repo.deTransportista(ctx, id)
	if errors.Is(err, errSinTransportista) {
		return ZonasTransportista{}, errNoEsTransportista
	}
	if err != nil {
		return ZonasTransportista{}, err
	}
	return ZonasTransportista{ZonaIDs: ids}, nil
}

// Reemplazar deja como zonas de trabajo exactamente las de in (RN-04). Devuelve
// no_es_transportista o zona_invalida. Escribe en la base.
func (s *Service) Reemplazar(ctx context.Context, id database.Identity, in ZonasTransportista) (ZonasTransportista, error) {
	err := s.repo.reemplazar(ctx, id, in.ZonaIDs)
	switch {
	case errors.Is(err, errSinTransportista):
		return ZonasTransportista{}, errNoEsTransportista
	case errors.Is(err, errZonaInexistente):
		return ZonasTransportista{}, errZonaInvalida
	case err != nil:
		return ZonasTransportista{}, err
	}
	return s.DeTransportista(ctx, id)
}
