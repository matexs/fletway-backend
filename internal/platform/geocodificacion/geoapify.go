package geocodificacion

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/matexs/fletway-backend/internal/platform/decimal"
	"github.com/matexs/fletway-backend/internal/platform/geoapify"
)

const (
	// radioZonaM es cuánto se aleja la búsqueda del centro de la zona. Alcanza
	// para cubrir un partido del conurbano o la Ciudad.
	radioZonaM = 20000
	// maxSugerencias es cuántas direcciones se proponen.
	maxSugerencias = 5
	// confianzaMinima descarta resultados que Geoapify no está seguro de haber
	// encontrado (0 a 1).
	confianzaMinima = 0.5
)

// Geoapify ubica direcciones reales con Geoapify (D-35), dentro de la zona
// elegida.
type Geoapify struct {
	Cliente *geoapify.Cliente
}

func zona(zonaNombre, provincia string) geoapify.Circulo {
	lat, lng, ok := CentroDeZona(zonaNombre, provincia)
	if !ok {
		return geoapify.Circulo{}
	}
	return geoapify.Circulo{Centro: geoapify.Punto{Lat: lat, Lng: lng}, RadioM: radioZonaM, Definir: true}
}

// Geocodificar implementa Geocodificador. Busca "dirección, zona, provincia,
// Argentina" cerca del centro de la zona. Devuelve ErrNoGeocodificable si no
// hay un resultado confiable.
func (g Geoapify) Geocodificar(ctx context.Context, direccion, zonaNombre, provincia string) (Punto, error) {
	texto := strings.Join([]string{direccion, zonaNombre, provincia, "Argentina"}, ", ")
	r, err := g.Cliente.Buscar(ctx, texto, zona(zonaNombre, provincia))
	if errors.Is(err, geoapify.ErrSinResultados) || (err == nil && r.Confianza < confianzaMinima) {
		return Punto{}, fmt.Errorf("%w: %q en %s", ErrNoGeocodificable, direccion, zonaNombre)
	}
	if err != nil {
		return Punto{}, fmt.Errorf("geocodificar: %w", err)
	}
	return Punto{
		Lat: decimal.MustParse(fmt.Sprintf("%.6f", r.Punto.Lat)),
		Lng: decimal.MustParse(fmt.Sprintf("%.6f", r.Punto.Lng)),
	}, nil
}

// Sugerir implementa Geocodificador con el autocompletado de Geoapify, dentro
// de la zona.
func (g Geoapify) Sugerir(ctx context.Context, texto, zonaNombre, provincia string) ([]Sugerencia, error) {
	rs, err := g.Cliente.Autocompletar(ctx, texto, zona(zonaNombre, provincia), maxSugerencias)
	if err != nil {
		return nil, fmt.Errorf("sugerir direcciones: %w", err)
	}
	out := make([]Sugerencia, 0, len(rs))
	for _, r := range rs {
		out = append(out, Sugerencia{Direccion: r.Linea1, Detalle: r.Linea2, Lat: r.Punto.Lat, Lng: r.Punto.Lng})
	}
	return out, nil
}
