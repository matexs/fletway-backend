// Package geoapify es el cliente HTTP de Geoapify (D-35): autocompletado y
// búsqueda de direcciones y ruteo por calles, sobre datos de OpenStreetMap. El
// plan gratis no pide tarjeta. La API key vive sólo en el backend.
package geoapify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// URLBase es la dirección de la API.
const URLBase = "https://api.geoapify.com"

// ErrSinResultados indica que la búsqueda no encontró nada.
var ErrSinResultados = errors.New("geoapify: sin resultados")

// Cliente llama a la API con una key. El valor cero no sirve: usar Nuevo.
type Cliente struct {
	key  string
	base string
	http *http.Client
}

// Nuevo crea el Cliente con la API key. base permite apuntar a un servidor de
// prueba; vacío usa URLBase.
func Nuevo(key, base string) *Cliente {
	if base == "" {
		base = URLBase
	}
	return &Cliente{key: key, base: base, http: &http.Client{Timeout: 8 * time.Second}}
}

// Punto son coordenadas en grados.
type Punto struct {
	Lat, Lng float64
}

// Circulo limita la búsqueda alrededor de un punto. El valor cero no limita.
type Circulo struct {
	Centro  Punto
	RadioM  int
	Definir bool
}

// Resultado es una dirección encontrada.
type Resultado struct {
	// Linea1 es la calle con la altura (o el nombre del lugar).
	Linea1 string
	// Linea2 es la localidad, el código postal y el país.
	Linea2 string
	Punto  Punto
	// Tipo es building, street, amenity, etc.
	Tipo string
	// Confianza va de 0 a 1.
	Confianza float64
}

type respuestaGeocode struct {
	Results []struct {
		AddressLine1 string  `json:"address_line1"`
		AddressLine2 string  `json:"address_line2"`
		Lat          float64 `json:"lat"`
		Lon          float64 `json:"lon"`
		ResultType   string  `json:"result_type"`
		Rank         struct {
			Confidence float64 `json:"confidence"`
		} `json:"rank"`
	} `json:"results"`
}

func (c *Cliente) geocode(ctx context.Context, ruta, texto string, zona Circulo, limite int) ([]Resultado, error) {
	q := url.Values{
		"text":   {texto},
		"lang":   {"es"},
		"limit":  {strconv.Itoa(limite)},
		"format": {"json"},
		"filter": {"countrycode:ar"},
	}
	if zona.Definir {
		q.Set("filter", fmt.Sprintf("circle:%f,%f,%d", zona.Centro.Lng, zona.Centro.Lat, zona.RadioM))
		q.Set("bias", fmt.Sprintf("proximity:%f,%f", zona.Centro.Lng, zona.Centro.Lat))
	}
	var r respuestaGeocode
	if err := c.get(ctx, ruta, q, &r); err != nil {
		return nil, err
	}
	out := make([]Resultado, 0, len(r.Results))
	for _, x := range r.Results {
		out = append(out, Resultado{
			Linea1: x.AddressLine1, Linea2: x.AddressLine2, Punto: Punto{Lat: x.Lat, Lng: x.Lon},
			Tipo: x.ResultType, Confianza: x.Rank.Confidence,
		})
	}
	return out, nil
}

// Autocompletar devuelve hasta limite direcciones que empiezan como texto,
// dentro de zona si está definida.
func (c *Cliente) Autocompletar(ctx context.Context, texto string, zona Circulo, limite int) ([]Resultado, error) {
	return c.geocode(ctx, "/v1/geocode/autocomplete", texto, zona, limite)
}

// Buscar devuelve la dirección que mejor coincide con texto, dentro de zona si
// está definida. Devuelve ErrSinResultados si no encuentra ninguna.
func (c *Cliente) Buscar(ctx context.Context, texto string, zona Circulo) (Resultado, error) {
	rs, err := c.geocode(ctx, "/v1/geocode/search", texto, zona, 1)
	if err != nil {
		return Resultado{}, err
	}
	if len(rs) == 0 {
		return Resultado{}, ErrSinResultados
	}
	return rs[0], nil
}

// Ruta es el trayecto por calles en auto.
type Ruta struct {
	DistanciaM float64
	DuracionS  float64
	// Trazado son los puntos del recorrido, en orden.
	Trazado []Punto
}

type respuestaRuta struct {
	Features []struct {
		Properties struct {
			Distance float64 `json:"distance"`
			Time     float64 `json:"time"`
		} `json:"properties"`
		Geometry struct {
			// MultiLineString: listas de [lng, lat].
			Coordinates [][][2]float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"features"`
}

// Rutear calcula el trayecto en auto de origen a destino. Devuelve
// ErrSinResultados si no hay ruta.
func (c *Cliente) Rutear(ctx context.Context, origen, destino Punto) (Ruta, error) {
	q := url.Values{
		"waypoints": {fmt.Sprintf("%f,%f|%f,%f", origen.Lat, origen.Lng, destino.Lat, destino.Lng)},
		"mode":      {"drive"},
	}
	var r respuestaRuta
	if err := c.get(ctx, "/v1/routing", q, &r); err != nil {
		return Ruta{}, err
	}
	if len(r.Features) == 0 {
		return Ruta{}, ErrSinResultados
	}
	f := r.Features[0]
	ruta := Ruta{DistanciaM: f.Properties.Distance, DuracionS: f.Properties.Time}
	for _, linea := range f.Geometry.Coordinates {
		for _, p := range linea {
			ruta.Trazado = append(ruta.Trazado, Punto{Lat: p[1], Lng: p[0]})
		}
	}
	return ruta, nil
}

func (c *Cliente) get(ctx context.Context, ruta string, q url.Values, destino any) error {
	q.Set("apiKey", c.key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+ruta+"?"+q.Encode(), nil)
	if err != nil {
		return fmt.Errorf("geoapify: armar pedido: %w", err)
	}
	res, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("geoapify: pedido a %s: %w", ruta, ctx.Err())
		}
		// El error de net/http incluye la URL con la key: no se propaga su texto.
		return fmt.Errorf("geoapify: pedido a %s falló", ruta)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, res.Body)
		return fmt.Errorf("geoapify: %s respondió %d", ruta, res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(destino); err != nil {
		return fmt.Errorf("geoapify: leer respuesta de %s: %w", ruta, err)
	}
	return nil
}
