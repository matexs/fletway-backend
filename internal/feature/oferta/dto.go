package oferta

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/matexs/fletway-backend/internal/platform/decimal"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// maxAyudantes es el tope de ayudantes por oferta (D-23): hasta acá cada
// ayudante acorta la operación.
const maxAyudantes = 3

var uuidValido = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// OfertaRequest es el cuerpo de POST /solicitudes/{id}/ofertas y de
// POST /solicitudes/{id}/ofertas/cotizar: con qué vehículo y cuántos ayudantes
// se haría el servicio. El precio lo calcula el sistema (RN-01).
type OfertaRequest struct {
	VehiculoID        string `json:"vehiculo_id"`
	CantidadAyudantes int    `json:"cantidad_ayudantes"`
}

// Validate controla la forma del pedido. Devuelve datos_invalidos con el
// problema de cada campo.
func (r *OfertaRequest) Validate() error {
	r.VehiculoID = strings.TrimSpace(r.VehiculoID)
	e := map[string]any{}
	if !uuidValido.MatchString(r.VehiculoID) {
		e["vehiculo_id"] = "elegí uno de tus vehículos"
	}
	if r.CantidadAyudantes < 0 || r.CantidadAyudantes > maxAyudantes {
		e["cantidad_ayudantes"] = "entre 0 y 3"
	}
	if len(e) > 0 {
		return &httpx.APIError{Status: http.StatusBadRequest, Code: "datos_invalidos",
			Message: "hay campos con valores inválidos", Details: e}
	}
	return nil
}

// DesgloseResponse es el detalle del precio (oferta_costo). Sólo lo ven el
// Transportista dueño y el Administrador (D-23).
type DesgloseResponse struct {
	DistanciaKm        decimal.Decimal `json:"distancia_km"`
	DuracionRutaH      decimal.Decimal `json:"duracion_ruta_h"`
	DuracionOperacionH decimal.Decimal `json:"duracion_operacion_h"`
	CostoLaboral       decimal.Decimal `json:"costo_laboral"`
	CostoVehiculo      decimal.Decimal `json:"costo_vehiculo"`
	CostosAdicionales  decimal.Decimal `json:"costos_adicionales"`
	CostoOperativo     decimal.Decimal `json:"costo_operativo"`
	MargenPct          decimal.Decimal `json:"margen_pct"`
	PrecioNeto         decimal.Decimal `json:"precio_neto"`
	PorcentajeComision decimal.Decimal `json:"porcentaje_comision"`
	IvaPct             decimal.Decimal `json:"iva_pct"`
}

// CotizacionResponse es el precio que tendría la oferta, sin guardarla.
type CotizacionResponse struct {
	CantidadViajes    int              `json:"cantidad_viajes"`
	CantidadAyudantes int              `json:"cantidad_ayudantes"`
	PrecioCalculado   decimal.Decimal  `json:"precio_calculado"`
	Desglose          DesgloseResponse `json:"desglose"`
}

// OfertaResponse es una oferta del Transportista, con su desglose.
type OfertaResponse struct {
	ID                string           `json:"id"`
	Estado            string           `json:"estado"`
	SolicitudID       string           `json:"solicitud_id"`
	SolicitudEstado   string           `json:"solicitud_estado"`
	FechaServicio     string           `json:"fecha_servicio_deseada"`
	OrigenZona        string           `json:"origen_zona_nombre"`
	DestinoZona       string           `json:"destino_zona_nombre"`
	VehiculoID        string           `json:"vehiculo_id"`
	VehiculoPatente   string           `json:"vehiculo_patente"`
	VehiculoTipo      string           `json:"vehiculo_tipo"`
	CantidadViajes    int              `json:"cantidad_viajes"`
	CantidadAyudantes int              `json:"cantidad_ayudantes"`
	PrecioCalculado   decimal.Decimal  `json:"precio_calculado"`
	Desglose          DesgloseResponse `json:"desglose"`
	CreadoEn          time.Time        `json:"creado_en"`
}

// OfertaParaCliente es una oferta como la ve el Cliente (RF-07): sin desglose
// ni patente (la patente se ve recién al aceptar).
type OfertaParaCliente struct {
	ID                string          `json:"id"`
	TransportistaID   string          `json:"transportista_id"`
	Transportista     string          `json:"transportista_nombre"`
	Calificacion      *float64        `json:"calificacion_promedio"`
	CantidadResenas   int             `json:"cantidad_resenas"`
	TasaCumplimiento  float64         `json:"tasa_cumplimiento"`
	VehiculoTipo      string          `json:"vehiculo_tipo"`
	CantidadViajes    int             `json:"cantidad_viajes"`
	CantidadAyudantes int             `json:"cantidad_ayudantes"`
	PrecioCalculado   decimal.Decimal `json:"precio_calculado"`
	CreadoEn          time.Time       `json:"creado_en"`
}

// OfertasDeSolicitudResponse es una página de las ofertas pendientes de una
// solicitud, ordenadas por score (RN-05). Sin ver_mas trae las 3 primeras.
type OfertasDeSolicitudResponse struct {
	// CantidadAyudantesSolicitados es lo que pidió el Cliente (orientativo,
	// D-20), para compararlo con los ayudantes de cada oferta.
	CantidadAyudantesSolicitados int                 `json:"cantidad_ayudantes_solicitados"`
	Total                        int                 `json:"total"`
	Ofertas                      []OfertaParaCliente `json:"ofertas"`
	// SiguienteCursor es la posición desde la que sigue "ver más", o null si no
	// hay más.
	SiguienteCursor *int `json:"siguiente_cursor"`
}

// ViajeConfirmadoResponse es el viaje que nace al aceptar una oferta. Recién
// acá el Cliente ve la patente.
type ViajeConfirmadoResponse struct {
	ID                  string          `json:"id"`
	Estado              string          `json:"estado"`
	SolicitudID         string          `json:"solicitud_id"`
	TransportistaID     string          `json:"transportista_id"`
	TransportistaNombre string          `json:"transportista_nombre"`
	VehiculoPatente     string          `json:"vehiculo_patente"`
	VehiculoMarcaModelo *string         `json:"vehiculo_marca_modelo"`
	MontoTotal          decimal.Decimal `json:"monto_total"`
	CantidadViajes      int             `json:"cantidad_viajes"`
	CantidadAyudantes   int             `json:"cantidad_ayudantes"`
	FechaServicio       string          `json:"fecha_servicio_deseada"`
	OrigenDireccion     string          `json:"origen_direccion"`
	DestinoDireccion    string          `json:"destino_direccion"`
	CreadoEn            time.Time       `json:"creado_en"`
}
