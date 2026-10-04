// Package ruteo calcula la distancia, la duración y el recorrido por calles del
// trayecto origen → destino de una solicitud: entran en el precio de cada
// oferta (RN-01, D-24) y se dibujan en el mapa. La interfaz Ruteador separa el
// proveedor: Geoapify (D-35) con calles reales, o una aproximación en línea
// recta para desarrollar sin API key.
package ruteo

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/matexs/fletway-backend/internal/platform/geoapify"
)

var (
	// ErrRutaNoDisponible indica que no se pudo obtener la ruta. La oferta se
	// rechaza: nunca se estima la distancia "a ojo" (D-24).
	ErrRutaNoDisponible = errors.New("ruteo: ruta no disponible")
	// ErrProveedor indica un proveedor desconocido, no implementado o no permitido
	// en el entorno.
	ErrProveedor = errors.New("ruteo: proveedor no disponible")
)

// Coordenada es un punto en grados decimales.
type Coordenada struct {
	Lat, Lng float64
}

// Ruta es el trayecto de ida origen → destino.
type Ruta struct {
	DistanciaKm float64
	DuracionH   float64
	// Trazado son los puntos del recorrido para dibujarlo, de origen a destino.
	Trazado []Coordenada
}

// Ruteador obtiene la ruta por calle entre dos puntos.
type Ruteador interface {
	// Calcular devuelve la ruta de ida de origen a destino, o
	// ErrRutaNoDisponible.
	Calcular(ctx context.Context, origen, destino Coordenada) (Ruta, error)
}

// Nuevo devuelve el Ruteador del proveedor pedido (RUTEO_PROVEEDOR), con caché.
// "geoapify" necesita la API key. En producción sólo se acepta "geoapify"
// (D-35), para que la aproximación nunca llegue a usuarios reales. "fijo" es
// sólo para tests y no se elige por configuración. Devuelve ErrProveedor en
// otro caso.
func Nuevo(proveedor, entorno, apiKey string) (Ruteador, error) {
	switch {
	case entorno == "production" && proveedor != "geoapify":
		return nil, fmt.Errorf("%w: en producción hace falta RUTEO_PROVEEDOR=geoapify", ErrProveedor)
	case proveedor == "geoapify" && apiKey == "":
		return nil, fmt.Errorf("%w: geoapify necesita GEOAPIFY_API_KEY", ErrProveedor)
	case proveedor == "geoapify":
		return ConCache(Geoapify{Cliente: geoapify.Nuevo(apiKey, "")}), nil
	case proveedor == "aproximado":
		return Aproximado{}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrProveedor, proveedor)
	}
}

const (
	// radioTierraKm es el radio medio de la Tierra.
	radioTierraKm = 6371.0
	// factorCalle pasa de la línea recta a una distancia por calle aproximada.
	factorCalle = 1.3
	// velocidadKmH es la velocidad media supuesta en zona urbana.
	velocidadKmH = 30.0
)

// Aproximado estima la ruta como la distancia en línea recta × 1,3, recorrida a
// 30 km/h (D-24). Sólo sirve para desarrollo local.
type Aproximado struct{}

// Calcular implementa Ruteador. No falla.
func (Aproximado) Calcular(_ context.Context, origen, destino Coordenada) (Ruta, error) {
	km := distanciaRectaKm(origen, destino) * factorCalle
	return Ruta{DistanciaKm: km, DuracionH: km / velocidadKmH, Trazado: []Coordenada{origen, destino}}, nil
}

// distanciaRectaKm es la distancia sobre la esfera (fórmula de haversine).
func distanciaRectaKm(a, b Coordenada) float64 {
	rad := func(g float64) float64 { return g * math.Pi / 180 }
	dLat := rad(b.Lat - a.Lat)
	dLng := rad(b.Lng - a.Lng)
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rad(a.Lat))*math.Cos(rad(b.Lat))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * radioTierraKm * math.Asin(math.Sqrt(h))
}

// Fijo devuelve siempre la misma ruta, o Err si no es nil. Sólo para tests.
type Fijo struct {
	Ruta Ruta
	Err  error
}

// Calcular implementa Ruteador.
func (f Fijo) Calcular(context.Context, Coordenada, Coordenada) (Ruta, error) {
	if f.Err != nil {
		return Ruta{}, f.Err
	}
	return f.Ruta, nil
}

// maxPuntosTrazado es cuántos puntos del recorrido se devuelven: alcanzan para
// dibujarlo en el mapa sin mandar miles.
const maxPuntosTrazado = 300

// Geoapify calcula la ruta en auto por calles con Geoapify (D-35).
type Geoapify struct {
	Cliente *geoapify.Cliente
}

// Calcular implementa Ruteador. Devuelve ErrRutaNoDisponible si Geoapify no
// encuentra una ruta.
func (g Geoapify) Calcular(ctx context.Context, origen, destino Coordenada) (Ruta, error) {
	r, err := g.Cliente.Rutear(ctx, geoapify.Punto(origen), geoapify.Punto(destino))
	if errors.Is(err, geoapify.ErrSinResultados) {
		return Ruta{}, ErrRutaNoDisponible
	}
	if err != nil {
		return Ruta{}, fmt.Errorf("%w: %w", ErrRutaNoDisponible, err)
	}
	trazado := make([]Coordenada, 0, len(r.Trazado))
	for _, p := range r.Trazado {
		trazado = append(trazado, Coordenada(p))
	}
	return Ruta{
		DistanciaKm: r.DistanciaM / 1000,
		DuracionH:   r.DuracionS / 3600,
		Trazado:     reducir(trazado, maxPuntosTrazado),
	}, nil
}

// reducir se queda con hasta limite puntos repartidos a lo largo del
// recorrido, siempre con el primero y el último.
func reducir(puntos []Coordenada, limite int) []Coordenada {
	if len(puntos) <= limite || limite < 2 {
		return puntos
	}
	out := make([]Coordenada, 0, limite)
	paso := float64(len(puntos)-1) / float64(limite-1)
	for i := range limite {
		out = append(out, puntos[int(math.Round(float64(i)*paso))])
	}
	return out
}

// maxCache es cuántas rutas se recuerdan; al llenarse se vacía. Las
// solicitudes no se editan, así que una ruta calculada no cambia.
const maxCache = 1000

type cache struct {
	interno Ruteador
	mu      sync.Mutex
	rutas   map[[2]Coordenada]Ruta
}

// ConCache envuelve un Ruteador para no repetir pedidos por el mismo trayecto
// (cotizar varias veces, abrir el mapa). Sólo guarda las rutas que salieron
// bien.
func ConCache(r Ruteador) Ruteador {
	return &cache{interno: r, rutas: map[[2]Coordenada]Ruta{}}
}

func (c *cache) Calcular(ctx context.Context, origen, destino Coordenada) (Ruta, error) {
	clave := [2]Coordenada{origen, destino}
	c.mu.Lock()
	r, ok := c.rutas[clave]
	c.mu.Unlock()
	if ok {
		return r, nil
	}
	r, err := c.interno.Calcular(ctx, origen, destino)
	if err != nil {
		return Ruta{}, err
	}
	c.mu.Lock()
	if len(c.rutas) >= maxCache {
		c.rutas = map[[2]Coordenada]Ruta{}
	}
	c.rutas[clave] = r
	c.mu.Unlock()
	return r, nil
}
