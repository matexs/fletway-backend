package catalogo

import "github.com/matexs/fletway-backend/internal/platform/decimal"

// ObjetoResponse es un objeto del catálogo con sus medidas y restricciones de
// carga (RN-08). Al publicar una solicitud, estos valores se copian a
// solicitud_objeto (módulo 6).
type ObjetoResponse struct {
	ID             string          `json:"id"`
	Nombre         string          `json:"nombre"`
	PesoEstimadoKg decimal.Decimal `json:"peso_estimado_kg"`
	// LargoM, AnchoM y AltoM son las medidas del objeto; AltoM es el eje vertical.
	LargoM decimal.Decimal `json:"largo_m"`
	AnchoM decimal.Decimal `json:"ancho_m"`
	AltoM  decimal.Decimal `json:"alto_m"`
	// RotacionHorizontal permite girarlo sobre la base; RotacionVertical,
	// acostarlo. Apilable indica si se le puede poner carga encima (RN-02).
	RotacionHorizontal bool `json:"rotacion_horizontal"`
	RotacionVertical   bool `json:"rotacion_vertical"`
	Apilable           bool `json:"apilable"`
}
