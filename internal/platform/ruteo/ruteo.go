// Package ruteo calcula la distancia y la duración del trayecto origen → destino
// de una solicitud, que entran en el precio de cada oferta (RN-01, D-24). La
// interfaz Ruteador separa el proveedor: en esta etapa no hay API de mapas
// conectada (D-17), así que en desarrollo se usa una aproximación en línea recta;
// el proveedor real (Google Distance Matrix) se integra más adelante.
package ruteo

import (
	"context"
	"errors"
	"fmt"
	"math"
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
}

// Ruteador obtiene la ruta por calle entre dos puntos.
type Ruteador interface {
	// Calcular devuelve la ruta de ida de origen a destino, o
	// ErrRutaNoDisponible.
	Calcular(ctx context.Context, origen, destino Coordenada) (Ruta, error)
}

// Nuevo devuelve el Ruteador del proveedor pedido (RUTEO_PROVEEDOR). En
// producción sólo se acepta "google" (que todavía no está implementado), para que
// la aproximación nunca llegue a usuarios reales. "fijo" es sólo para tests y no
// se elige por configuración. Devuelve ErrProveedor en otro caso.
func Nuevo(proveedor, entorno string) (Ruteador, error) {
	switch {
	case entorno == "production" && proveedor != "google":
		return nil, fmt.Errorf("%w: en producción hace falta RUTEO_PROVEEDOR=google", ErrProveedor)
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
	return Ruta{DistanciaKm: km, DuracionH: km / velocidadKmH}, nil
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
