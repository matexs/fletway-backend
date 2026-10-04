package perfil

import (
	"context"
	"errors"

	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

var errNoEncontradoAPI = httpx.NotFound("transportista_no_encontrado", "no existe el Transportista")

// Service lee perfiles públicos de Transportistas (RF-11).
type Service struct {
	repo *repository
}

// NewService crea el Service sobre la base dada.
func NewService(db *database.DB) *Service {
	return &Service{repo: &repository{db: db}}
}

// Perfil devuelve el perfil público de un Transportista habilitado, visible
// para cualquier usuario autenticado. Devuelve transportista_no_encontrado si
// no existe o todavía no está habilitado.
func (s *Service) Perfil(ctx context.Context, id database.Identity, transportistaID string) (PerfilResponse, error) {
	p, err := s.repo.perfil(ctx, id, transportistaID)
	if errors.Is(err, errNoEncontrado) {
		return PerfilResponse{}, errNoEncontradoAPI
	}
	return p, err
}
