package matchmaking

import (
	"time"

	"github.com/matexs/fletway-backend/internal/platform/decimal"
)

// SolicitudCompatible es una solicitud que el Transportista puede ofertar (D-21),
// resumida para el listado. No tiene montos: el precio se calcula al ofertar
// (RN-01, módulo 8). El detalle completo sale de GET /api/solicitudes/{id}.
type SolicitudCompatible struct {
	ID                           string          `json:"id"`
	FechaServicioDeseada         string          `json:"fecha_servicio_deseada"`
	FranjaHorariaInicio          *string         `json:"franja_horaria_inicio"`
	FranjaHorariaFin             *string         `json:"franja_horaria_fin"`
	OrigenZonaNombre             string          `json:"origen_zona_nombre"`
	DestinoZonaNombre            string          `json:"destino_zona_nombre"`
	CantidadObjetos              int             `json:"cantidad_objetos"`
	PesoTotalKg                  decimal.Decimal `json:"peso_total_kg"`
	VolumenTotalM3               decimal.Decimal `json:"volumen_total_m3"`
	CantidadAyudantesSolicitados int             `json:"cantidad_ayudantes_solicitados"`
	CreadoEn                     time.Time       `json:"creado_en"`
}
