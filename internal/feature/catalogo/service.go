package catalogo

import (
	"context"

	"github.com/matexs/fletway-backend/internal/platform/database"
)

// Service lee el catálogo de objetos comunes (RN-08).
type Service struct {
	repo *repository
}

// NewService crea el Service sobre la base dada.
func NewService(db *database.DB) *Service {
	return &Service{repo: &repository{db: db}}
}

// Objetos devuelve el catálogo completo, ordenado por nombre. Lo puede leer
// cualquier usuario autenticado.
func (s *Service) Objetos(ctx context.Context, id database.Identity) ([]ObjetoResponse, error) {
	return s.repo.objetos(ctx, id)
}
