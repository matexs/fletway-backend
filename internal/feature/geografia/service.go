package geografia

import (
	"context"
	"errors"

	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

var (
	errNoEsTransportista = httpx.Forbidden("no_es_transportista",
		"sólo un Transportista registrado puede elegir zonas de trabajo")
	errZonaInvalida = httpx.BadRequest("zona_invalida", "alguna de las zonas no existe")
)

// Service lee el catálogo de zonas y guarda las del Transportista (RN-04).
type Service struct {
	repo *repository
}

// NewService crea el Service sobre la base dada.
func NewService(db *database.DB) *Service {
	return &Service{repo: &repository{db: db}}
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
