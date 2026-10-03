// Package geocodificacion convierte una dirección escrita a mano en coordenadas
// (D-20). La interfaz Geocodificador separa el proveedor: en esta etapa no hay
// API de mapas conectada (D-17), así que en desarrollo se usa una aproximación
// por zona; el proveedor real (Google Geocoding) se integra más adelante.
package geocodificacion

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"

	"github.com/matexs/fletway-backend/internal/platform/decimal"
)

var (
	// ErrNoGeocodificable indica que no se pudo ubicar la dirección.
	ErrNoGeocodificable = errors.New("geocodificacion: no se pudo ubicar la dirección")
	// ErrProveedor indica un proveedor desconocido, no implementado o no permitido
	// en el entorno.
	ErrProveedor = errors.New("geocodificacion: proveedor no disponible")
)

// Punto son coordenadas con 6 decimales (numeric(9,6) de solicitud).
type Punto struct {
	Lat decimal.Decimal
	Lng decimal.Decimal
}

// Geocodificador ubica una dirección dentro de su zona.
type Geocodificador interface {
	// Geocodificar devuelve las coordenadas de direccion, que está en la zona
	// zonaNombre de provincia. Devuelve ErrNoGeocodificable si no la ubica.
	Geocodificar(ctx context.Context, direccion, zonaNombre, provincia string) (Punto, error)
}

// Nuevo devuelve el Geocodificador del proveedor pedido. En producción sólo se
// acepta "google" (que todavía no está implementado), para que la aproximación
// nunca llegue a usuarios reales. Devuelve ErrProveedor en otro caso.
func Nuevo(proveedor, entorno string) (Geocodificador, error) {
	switch {
	case entorno == "production" && proveedor != "google":
		return nil, fmt.Errorf("%w: en producción hace falta GEOCODIFICADOR_PROVEEDOR=google", ErrProveedor)
	case proveedor == "aproximado":
		return Aproximado{}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrProveedor, proveedor)
	}
}

// centros son coordenadas aproximadas del centro de cada zona piloto (migración
// 0008), por "nombre|provincia".
var centros = map[string][2]float64{
	"Ciudad Autónoma de Buenos Aires|CABA": {-34.6037, -58.3816},
	"Campana|Buenos Aires":                 {-34.1633, -58.9592},
	"Zárate|Buenos Aires":                  {-34.0981, -59.0286},
	"Escobar|Buenos Aires":                 {-34.3467, -58.7953},
	"Pilar|Buenos Aires":                   {-34.4587, -58.9142},
	"Tigre|Buenos Aires":                   {-34.4264, -58.5796},
	"San Fernando|Buenos Aires":            {-34.4417, -58.5594},
	"San Isidro|Buenos Aires":              {-34.4708, -58.5286},
	"Vicente López|Buenos Aires":           {-34.5260, -58.4740},
	"San Martín|Buenos Aires":              {-34.5750, -58.5370},
	"Tres de Febrero|Buenos Aires":         {-34.6047, -58.5636},
	"San Miguel|Buenos Aires":              {-34.5423, -58.7124},
	"Malvinas Argentinas|Buenos Aires":     {-34.4900, -58.7100},
	"José C. Paz|Buenos Aires":             {-34.5150, -58.7680},
}

// desplazamientoMax es el corrimiento máximo desde el centro, en grados
// (unos 1,5 km).
const desplazamientoMax = 0.015

// Aproximado ubica la dirección cerca del centro de su zona, corrida en forma
// determinística según el texto: la misma dirección siempre da el mismo punto y
// dos direcciones distintas de una zona no quedan a distancia cero. Sólo sirve
// para desarrollo (las distancias son orientativas).
type Aproximado struct{}

// Geocodificar implementa Geocodificador. Devuelve ErrNoGeocodificable si la zona
// no está entre las piloto.
func (Aproximado) Geocodificar(_ context.Context, direccion, zonaNombre, provincia string) (Punto, error) {
	c, ok := centros[zonaNombre+"|"+provincia]
	if !ok {
		return Punto{}, fmt.Errorf("%w: zona %q sin coordenadas aproximadas", ErrNoGeocodificable, zonaNombre)
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(direccion))
	v := h.Sum64()
	// Dos valores en [-1, 1] sacados de mitades distintas del hash.
	dx := float64(v&0xffffffff)/float64(0xffffffff)*2 - 1
	dy := float64(v>>32)/float64(0xffffffff)*2 - 1
	return Punto{
		Lat: decimal.MustParse(fmt.Sprintf("%.6f", c[0]+dy*desplazamientoMax)),
		Lng: decimal.MustParse(fmt.Sprintf("%.6f", c[1]+dx*desplazamientoMax)),
	}, nil
}
