// Package geocodificacion convierte una dirección en coordenadas (D-20) y
// sugiere direcciones mientras el Cliente escribe. La interfaz Geocodificador
// separa el proveedor: Geoapify (D-35) con datos reales, o una aproximación por
// zona para desarrollar sin API key.
package geocodificacion

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"

	"github.com/matexs/fletway-backend/internal/platform/decimal"
	"github.com/matexs/fletway-backend/internal/platform/geoapify"
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

// Sugerencia es una dirección propuesta mientras el Cliente escribe.
type Sugerencia struct {
	// Direccion es la calle con la altura, lista para guardar en la solicitud.
	Direccion string `json:"direccion"`
	// Detalle es la localidad y el código postal, para distinguir sugerencias.
	Detalle string  `json:"detalle"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
}

// Geocodificador ubica una dirección dentro de su zona.
type Geocodificador interface {
	// Geocodificar devuelve las coordenadas de direccion, que está en la zona
	// zonaNombre de provincia. Devuelve ErrNoGeocodificable si no la ubica.
	Geocodificar(ctx context.Context, direccion, zonaNombre, provincia string) (Punto, error)
	// Sugerir devuelve direcciones de la zona que empiezan como texto, para
	// autocompletar. Puede devolver una lista vacía.
	Sugerir(ctx context.Context, texto, zonaNombre, provincia string) ([]Sugerencia, error)
}

// Nuevo devuelve el Geocodificador del proveedor pedido. "geoapify" necesita
// la API key. En producción sólo se acepta "geoapify" (D-35), para que la
// aproximación nunca llegue a usuarios reales. Devuelve ErrProveedor en otro
// caso.
func Nuevo(proveedor, entorno, apiKey string) (Geocodificador, error) {
	switch {
	case entorno == "production" && proveedor != "geoapify":
		return nil, fmt.Errorf("%w: en producción hace falta GEOCODIFICADOR_PROVEEDOR=geoapify", ErrProveedor)
	case proveedor == "geoapify" && apiKey == "":
		return nil, fmt.Errorf("%w: geoapify necesita GEOAPIFY_API_KEY", ErrProveedor)
	case proveedor == "geoapify":
		return Geoapify{Cliente: geoapify.Nuevo(apiKey, "")}, nil
	case proveedor == "aproximado":
		return Aproximado{}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrProveedor, proveedor)
	}
}

// centros son coordenadas aproximadas del centro de cada zona piloto (migración
// 0008), por "nombre|provincia". Geoapify las usa para limitar la búsqueda a la
// zona y Aproximado para inventar un punto cercano.
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

// Sugerir implementa Geocodificador: sin datos de mapas, la única sugerencia es
// lo que escribió el Cliente.
func (Aproximado) Sugerir(_ context.Context, texto, zonaNombre, _ string) ([]Sugerencia, error) {
	return []Sugerencia{{Direccion: texto, Detalle: zonaNombre}}, nil
}

// CentroDeZona devuelve el centro aproximado de una zona piloto, si se conoce.
func CentroDeZona(zonaNombre, provincia string) (lat, lng float64, ok bool) {
	c, ok := centros[zonaNombre+"|"+provincia]
	return c[0], c[1], ok
}
