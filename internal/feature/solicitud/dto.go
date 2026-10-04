package solicitud

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/matexs/fletway-backend/internal/platform/decimal"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// Límites de forma del alta. Los de pisos, objetos y cantidades son cotas de
// sentido común contra errores de tipeo, no reglas de negocio.
const (
	maxAyudantes = 3
	maxObjetos   = 50
	maxCantidad  = 100
	maxPisos     = 60
)

var (
	uuidValido = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	horaValida = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
)

// errores acumula los problemas de validación por campo (details).
type errores map[string]any

func (e errores) error() error {
	if len(e) == 0 {
		return nil
	}
	return &httpx.APIError{Status: http.StatusBadRequest, Code: "datos_invalidos",
		Message: "hay campos con valores inválidos", Details: e}
}

func (e errores) rango(campo string, d decimal.Decimal, minExclusivo bool, min, max string, escala int) {
	lo, hi := decimal.MustParse(min), decimal.MustParse(max)
	switch {
	case d.Escala() < 0 || d.Escala() > escala:
		e[campo] = "admite hasta " + strconv.Itoa(escala) + " decimales"
	case minExclusivo && d.Cmp(lo) <= 0:
		e[campo] = "tiene que ser mayor que " + min
	case !minExclusivo && d.Cmp(lo) < 0:
		e[campo] = "tiene que ser mayor o igual que " + min
	case d.Cmp(hi) > 0:
		e[campo] = "no puede superar " + max
	}
}

func (e errores) texto(campo, v string, min, max int) {
	n := utf8.RuneCountInString(v)
	if n < min || n > max {
		e[campo] = fmt.Sprintf("entre %d y %d caracteres", min, max)
	}
}

// PuntoRequest es el origen o el destino del traslado, con sus condiciones de
// acceso (RN-01: escaleras, ascensor y distancia a pie influyen en el precio).
type PuntoRequest struct {
	ZonaID    string `json:"zona_id"`
	Direccion string `json:"direccion"`
	// Pisos por escalera hasta el lugar (0 = planta baja).
	Pisos              int             `json:"pisos"`
	AscensorUtilizable bool            `json:"ascensor_utilizable"`
	DistanciaVehiculoM decimal.Decimal `json:"distancia_vehiculo_m"`
}

func (p *PuntoRequest) validar(e errores, prefijo string) {
	p.ZonaID = strings.TrimSpace(p.ZonaID)
	p.Direccion = strings.TrimSpace(p.Direccion)
	if !uuidValido.MatchString(p.ZonaID) {
		e[prefijo+".zona_id"] = "elegí una zona del catálogo"
	}
	e.texto(prefijo+".direccion", p.Direccion, 5, 200)
	if p.Pisos < 0 || p.Pisos > maxPisos {
		e[prefijo+".pisos"] = fmt.Sprintf("entre 0 y %d", maxPisos)
	}
	e.rango(prefijo+".distancia_vehiculo_m", p.DistanciaVehiculoM, false, "0", "9999.9", 1)
}

// ObjetoRequest es un objeto a trasladar: del catálogo (sólo objeto_id y
// cantidad; peso, medidas y restricciones se copian del catálogo, RN-08) o
// cargado a mano (nombre_personalizado con peso y medidas).
type ObjetoRequest struct {
	ObjetoID            *string          `json:"objeto_id"`
	NombrePersonalizado *string          `json:"nombre_personalizado"`
	Cantidad            int              `json:"cantidad"`
	PesoUnitarioKg      *decimal.Decimal `json:"peso_unitario_kg"`
	LargoM              *decimal.Decimal `json:"largo_m"`
	AnchoM              *decimal.Decimal `json:"ancho_m"`
	AltoM               *decimal.Decimal `json:"alto_m"`
	// Restricciones de un objeto manual; si faltan valen true (como en la tabla).
	RotacionHorizontal *bool `json:"rotacion_horizontal"`
	RotacionVertical   *bool `json:"rotacion_vertical"`
	Apilable           *bool `json:"apilable"`
}

// EsDelCatalogo indica si el objeto viene del catálogo.
func (o *ObjetoRequest) EsDelCatalogo() bool { return o.ObjetoID != nil }

func (o *ObjetoRequest) validar(e errores, prefijo string) {
	if o.Cantidad < 1 || o.Cantidad > maxCantidad {
		e[prefijo+".cantidad"] = fmt.Sprintf("entre 1 y %d", maxCantidad)
	}
	manual := o.NombrePersonalizado != nil || o.PesoUnitarioKg != nil || o.LargoM != nil ||
		o.AnchoM != nil || o.AltoM != nil || o.RotacionHorizontal != nil || o.RotacionVertical != nil ||
		o.Apilable != nil
	switch {
	case o.ObjetoID != nil && manual:
		e[prefijo] = "un objeto del catálogo lleva sólo objeto_id y cantidad"
	case o.ObjetoID != nil:
		if !uuidValido.MatchString(*o.ObjetoID) {
			e[prefijo+".objeto_id"] = "elegí un objeto del catálogo"
		}
	default:
		nombre := ""
		if o.NombrePersonalizado != nil {
			nombre = strings.TrimSpace(*o.NombrePersonalizado)
			o.NombrePersonalizado = &nombre
		}
		e.texto(prefijo+".nombre_personalizado", nombre, 2, 80)
		medidas := []struct {
			campo, max string
			v          *decimal.Decimal
		}{
			{"peso_unitario_kg", "2000", o.PesoUnitarioKg},
			{"largo_m", "10", o.LargoM},
			{"ancho_m", "10", o.AnchoM},
			{"alto_m", "10", o.AltoM},
		}
		for _, m := range medidas {
			if m.v == nil {
				e[prefijo+"."+m.campo] = "es obligatorio en un objeto cargado a mano"
				continue
			}
			e.rango(prefijo+"."+m.campo, *m.v, true, "0", m.max, 2)
		}
	}
}

// CrearSolicitudRequest publica una necesidad de traslado (RF-06). No lleva
// ningún monto: no hay cotización estimada (RN-01).
type CrearSolicitudRequest struct {
	Origen  PuntoRequest `json:"origen"`
	Destino PuntoRequest `json:"destino"`
	// FechaServicioDeseada en formato AAAA-MM-DD.
	FechaServicioDeseada string `json:"fecha_servicio_deseada"`
	// Franja opcional en formato HH:MM; van las dos o ninguna.
	FranjaHorariaInicio          *string         `json:"franja_horaria_inicio"`
	FranjaHorariaFin             *string         `json:"franja_horaria_fin"`
	CantidadAyudantesSolicitados int             `json:"cantidad_ayudantes_solicitados"`
	Objetos                      []ObjetoRequest `json:"objetos"`

	fecha time.Time
}

// Fecha devuelve la fecha del servicio ya validada.
func (r *CrearSolicitudRequest) Fecha() time.Time { return r.fecha }

// Validate controla la forma de todo el request y devuelve todos los problemas
// juntos en un *httpx.APIError datos_invalidos. Que la fecha no sea pasada lo
// controla el Service, que conoce la fecha de hoy.
func (r *CrearSolicitudRequest) Validate() error {
	e := errores{}
	r.Origen.validar(e, "origen")
	r.Destino.validar(e, "destino")
	r.validarFechaYFranja(e)
	if r.CantidadAyudantesSolicitados < 0 || r.CantidadAyudantesSolicitados > maxAyudantes {
		e["cantidad_ayudantes_solicitados"] = fmt.Sprintf("entre 0 y %d", maxAyudantes)
	}
	if len(r.Objetos) == 0 || len(r.Objetos) > maxObjetos {
		e["objetos"] = fmt.Sprintf("entre 1 y %d objetos", maxObjetos)
	}
	for i := range r.Objetos {
		r.Objetos[i].validar(e, fmt.Sprintf("objetos[%d]", i))
	}
	return e.error()
}

func (r *CrearSolicitudRequest) validarFechaYFranja(e errores) {
	f, err := time.Parse(time.DateOnly, r.FechaServicioDeseada)
	if err != nil {
		e["fecha_servicio_deseada"] = "formato AAAA-MM-DD"
	}
	r.fecha = f
	validarFranja(e, r.FranjaHorariaInicio, r.FranjaHorariaFin)
}

func validarFranja(e errores, inicio, fin *string) {
	switch {
	case inicio == nil && fin == nil:
	case inicio == nil || fin == nil:
		e["franja_horaria"] = "cargá el inicio y el fin, o ninguno"
	case !horaValida.MatchString(*inicio) || !horaValida.MatchString(*fin):
		e["franja_horaria"] = "formato HH:MM"
	case *inicio >= *fin:
		e["franja_horaria"] = "el inicio tiene que ser anterior al fin"
	}
}

// RepublicarRequest es la fecha nueva para republicar una solicitud vencida
// (D-20). La franja es opcional, como en el alta.
type RepublicarRequest struct {
	FechaServicioDeseada string  `json:"fecha_servicio_deseada"`
	FranjaHorariaInicio  *string `json:"franja_horaria_inicio"`
	FranjaHorariaFin     *string `json:"franja_horaria_fin"`

	fecha time.Time
}

// Validate controla la forma; devuelve un *httpx.APIError datos_invalidos.
func (r *RepublicarRequest) Validate() error {
	e := errores{}
	f, err := time.Parse(time.DateOnly, r.FechaServicioDeseada)
	if err != nil {
		e["fecha_servicio_deseada"] = "formato AAAA-MM-DD"
	}
	r.fecha = f
	validarFranja(e, r.FranjaHorariaInicio, r.FranjaHorariaFin)
	return e.error()
}

// PuntoResponse es el origen o el destino de una solicitud.
type PuntoResponse struct {
	ZonaID             string          `json:"zona_id"`
	ZonaNombre         string          `json:"zona_nombre"`
	Direccion          string          `json:"direccion"`
	Pisos              int             `json:"pisos"`
	AscensorUtilizable bool            `json:"ascensor_utilizable"`
	DistanciaVehiculoM decimal.Decimal `json:"distancia_vehiculo_m"`
}

// ObjetoResponse es un objeto de la solicitud con los valores copiados al
// publicar (RN-08).
type ObjetoResponse struct {
	ID                 string          `json:"id"`
	ObjetoID           *string         `json:"objeto_id"`
	Nombre             string          `json:"nombre"`
	Cantidad           int             `json:"cantidad"`
	PesoUnitarioKg     decimal.Decimal `json:"peso_unitario_kg"`
	LargoM             decimal.Decimal `json:"largo_m"`
	AnchoM             decimal.Decimal `json:"ancho_m"`
	AltoM              decimal.Decimal `json:"alto_m"`
	RotacionHorizontal bool            `json:"rotacion_horizontal"`
	RotacionVertical   bool            `json:"rotacion_vertical"`
	Apilable           bool            `json:"apilable"`
}

// SolicitudResponse es el detalle de una solicitud. Ningún campo es un monto
// (RN-01: el único precio es el de cada oferta).
type SolicitudResponse struct {
	ID string `json:"id"`
	// Estado es publicada, vencida (publicada con la fecha ya pasada, calculado
	// al leer, D-20), asignada, cancelada o expirada.
	Estado                       string           `json:"estado"`
	FechaServicioDeseada         string           `json:"fecha_servicio_deseada"`
	FranjaHorariaInicio          *string          `json:"franja_horaria_inicio"`
	FranjaHorariaFin             *string          `json:"franja_horaria_fin"`
	Origen                       PuntoResponse    `json:"origen"`
	Destino                      PuntoResponse    `json:"destino"`
	CantidadAyudantesSolicitados int              `json:"cantidad_ayudantes_solicitados"`
	Objetos                      []ObjetoResponse `json:"objetos"`
	CreadoEn                     time.Time        `json:"creado_en"`
}

// SolicitudResumen es una solicitud en el listado del Cliente.
type SolicitudResumen struct {
	ID                   string    `json:"id"`
	Estado               string    `json:"estado"`
	FechaServicioDeseada string    `json:"fecha_servicio_deseada"`
	FranjaHorariaInicio  *string   `json:"franja_horaria_inicio"`
	FranjaHorariaFin     *string   `json:"franja_horaria_fin"`
	OrigenZonaNombre     string    `json:"origen_zona_nombre"`
	DestinoZonaNombre    string    `json:"destino_zona_nombre"`
	CantidadObjetos      int       `json:"cantidad_objetos"`
	CreadoEn             time.Time `json:"creado_en"`
}

// PuntoMapa son coordenadas para el mapa, en grados.
type PuntoMapa struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// RutaResponse es el recorrido de una solicitud para dibujarlo (D-35):
// extremos, distancia y tiempo de manejo de ida, y los puntos del trayecto por
// calles.
type RutaResponse struct {
	Origen      PuntoMapa   `json:"origen"`
	Destino     PuntoMapa   `json:"destino"`
	DistanciaKm float64     `json:"distancia_km"`
	DuracionMin int         `json:"duracion_min"`
	Trazado     []PuntoMapa `json:"trazado"`
}
