package vehiculo

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/matexs/fletway-backend/internal/platform/decimal"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// Formatos de patente argentina: AAA999 (anterior) y AA999AA (Mercosur).
var patenteValida = regexp.MustCompile(`^([A-Z]{3}[0-9]{3}|[A-Z]{2}[0-9]{3}[A-Z]{2})$`)

// errores acumula los problemas de validación por campo, para devolverlos todos
// juntos en details.
type errores map[string]any

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

func (e errores) error() error {
	if len(e) == 0 {
		return nil
	}
	return &httpx.APIError{
		Status:  http.StatusBadRequest,
		Code:    "datos_invalidos",
		Message: "hay campos con valores inválidos",
		Details: e,
	}
}

// TipoVehiculoResponse es un tipo de vehículo con sus medidas estándar de
// referencia (D-32): la app las propone al registrar un vehículo.
type TipoVehiculoResponse struct {
	ID                   string          `json:"id"`
	Nombre               string          `json:"nombre"`
	LargoEstandarM       decimal.Decimal `json:"largo_estandar_m"`
	AnchoEstandarM       decimal.Decimal `json:"ancho_estandar_m"`
	AltoEstandarM        decimal.Decimal `json:"alto_estandar_m"`
	PesoMaximoEstandarKg decimal.Decimal `json:"peso_maximo_estandar_kg"`
}

// CrearVehiculoRequest registra un vehículo del Transportista (RF-18). Las
// medidas son las útiles de la caja y el peso es la carga útil: son las que usa
// el cálculo de viajes (RN-02).
type CrearVehiculoRequest struct {
	TipoVehiculoID string          `json:"tipo_vehiculo_id"`
	Patente        string          `json:"patente"`
	Marca          *string         `json:"marca"`
	Modelo         *string         `json:"modelo"`
	LargoUtilM     decimal.Decimal `json:"largo_util_m"`
	AnchoUtilM     decimal.Decimal `json:"ancho_util_m"`
	AltoUtilM      decimal.Decimal `json:"alto_util_m"`
	PesoMaximoKg   decimal.Decimal `json:"peso_maximo_kg"`
}

// Validate normaliza la patente (mayúsculas, sin espacios ni guiones) y controla
// la forma. Devuelve un *httpx.APIError datos_invalidos con el detalle por campo.
func (r *CrearVehiculoRequest) Validate() error {
	e := errores{}
	r.Patente = strings.ToUpper(strings.NewReplacer(" ", "", "-", "", ".", "").Replace(r.Patente))
	if !patenteValida.MatchString(r.Patente) {
		e["patente"] = "formato AAA999 o AA999AA"
	}
	if strings.TrimSpace(r.TipoVehiculoID) == "" {
		e["tipo_vehiculo_id"] = "es obligatorio"
	}
	r.Marca, r.Modelo = recortar(r.Marca), recortar(r.Modelo)
	for campo, v := range map[string]*string{"marca": r.Marca, "modelo": r.Modelo} {
		if v != nil && len([]rune(*v)) > 60 {
			e[campo] = "no puede superar los 60 caracteres"
		}
	}
	e.rango("largo_util_m", r.LargoUtilM, true, "0", "20", 2)
	e.rango("ancho_util_m", r.AnchoUtilM, true, "0", "3", 2)
	e.rango("alto_util_m", r.AltoUtilM, true, "0", "5", 2)
	e.rango("peso_maximo_kg", r.PesoMaximoKg, true, "0", "40000", 2)
	return e.error()
}

func recortar(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// VehiculoResponse es un vehículo del Transportista.
type VehiculoResponse struct {
	ID                 string          `json:"id"`
	TipoVehiculoID     string          `json:"tipo_vehiculo_id"`
	TipoVehiculoNombre string          `json:"tipo_vehiculo_nombre"`
	Patente            string          `json:"patente"`
	Marca              *string         `json:"marca"`
	Modelo             *string         `json:"modelo"`
	LargoUtilM         decimal.Decimal `json:"largo_util_m"`
	AnchoUtilM         decimal.Decimal `json:"ancho_util_m"`
	AltoUtilM          decimal.Decimal `json:"alto_util_m"`
	PesoMaximoKg       decimal.Decimal `json:"peso_maximo_kg"`
	Activo             bool            `json:"activo"`
	CreadoEn           time.Time       `json:"creado_en"`
}

// ActivoRequest activa o desactiva un vehículo.
type ActivoRequest struct {
	Activo *bool `json:"activo"`
}

// Validate exige el campo activo.
func (r *ActivoRequest) Validate() error {
	if r.Activo == nil {
		return httpx.BadRequest("datos_incompletos", "falta activo")
	}
	return nil
}
