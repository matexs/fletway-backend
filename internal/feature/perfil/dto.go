package perfil

import (
	"regexp"
	"time"
)

var uuidValido = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ResenaResponse es una reseña pública de un viaje terminado.
type ResenaResponse struct {
	Calificacion int       `json:"calificacion"`
	Mensaje      *string   `json:"mensaje"`
	Cliente      string    `json:"cliente_nombre"`
	CreadoEn     time.Time `json:"creado_en"`
}

// PerfilResponse es el perfil público de un Transportista habilitado (RF-11).
type PerfilResponse struct {
	ID               string   `json:"id"`
	Nombre           string   `json:"nombre"`
	FormaTrabajo     *string  `json:"forma_trabajo"`
	Calificacion     *float64 `json:"calificacion_promedio"`
	CantidadResenas  int      `json:"cantidad_resenas"`
	TasaCumplimiento float64  `json:"tasa_cumplimiento"`
	// Zonas son los nombres de sus zonas de trabajo.
	Zonas []string `json:"zonas"`
	// TiposVehiculo son los tipos de sus vehículos activos, sin patentes.
	TiposVehiculo []string `json:"tipos_vehiculo"`
	// Resenas son las 20 más recientes.
	Resenas []ResenaResponse `json:"resenas"`
	DesdeEn time.Time        `json:"en_fletway_desde"`
}
