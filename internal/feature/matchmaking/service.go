package matchmaking

import (
	"context"

	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

var errNoEsTransportista = httpx.Forbidden("no_es_transportista",
	"sólo un Transportista registrado ve solicitudes compatibles")

// Service resuelve qué solicitudes ve cada Transportista y los avisa al publicar
// (RN-04, RN-05, D-21, D-22).
type Service struct {
	repo *repository
}

// NewService crea el Service sobre la base dada.
func NewService(db *database.DB) *Service {
	return &Service{repo: &repository{db: db}}
}

// Compatibles lista las solicitudes que el Transportista puede ofertar (D-21):
// publicadas y no vencidas, de sus zonas, con algún vehículo activo con
// capacidad, y sólo si está habilitado, disponible y sin veto. Devuelve
// no_es_transportista.
func (s *Service) Compatibles(ctx context.Context, id database.Identity) ([]SolicitudCompatible, error) {
	es, err := s.repo.esTransportista(ctx, id)
	if err != nil {
		return nil, err
	}
	if !es {
		return nil, errNoEsTransportista
	}
	return s.repo.compatibles(ctx, id)
}

// NotificarCompatibles crea la notificación in-app solicitud_compatible para cada
// Transportista compatible con la solicitud (RN-05, D-22). La identidad es la del
// Cliente dueño. Es idempotente: se puede reintentar sin duplicar avisos.
// Escribe en la base.
func (s *Service) NotificarCompatibles(ctx context.Context, id database.Identity, solicitudID string) error {
	_, err := s.repo.notificar(ctx, id, solicitudID)
	return err
}
