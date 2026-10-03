package ruteo

import (
	"context"
	"errors"
	"math"
	"testing"
)

func TestNuevo(t *testing.T) {
	t.Parallel()
	casos := []struct {
		name      string
		proveedor string
		entorno   string
		wantErr   bool
	}{
		{"aproximado en desarrollo", "aproximado", "development", false},
		{"aproximado en producción", "aproximado", "production", true},
		{"google todavía no implementado", "google", "development", true},
		{"desconocido", "otro", "development", true},
		{"fijo no se elige por configuración", "fijo", "development", true},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Nuevo(tc.proveedor, tc.entorno)
			if tc.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil && !errors.Is(err, ErrProveedor) {
				t.Fatalf("err = %v, want ErrProveedor", err)
			}
		})
	}
}

func TestAproximado(t *testing.T) {
	t.Parallel()
	casos := []struct {
		name            string
		origen, destino Coordenada
		wantKm          float64
	}{
		{"mismo punto", Coordenada{-34.6, -58.4}, Coordenada{-34.6, -58.4}, 0},
		// Un grado de latitud son unos 111,19 km en línea recta.
		{"un grado de latitud", Coordenada{-34, -58}, Coordenada{-35, -58}, 111.19 * 1.3},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r, err := Aproximado{}.Calcular(context.Background(), tc.origen, tc.destino)
			if err != nil {
				t.Fatal(err)
			}
			if math.Abs(r.DistanciaKm-tc.wantKm) > 0.1 {
				t.Fatalf("DistanciaKm = %v, want %v", r.DistanciaKm, tc.wantKm)
			}
			if math.Abs(r.DuracionH-r.DistanciaKm/30) > 1e-9 {
				t.Fatalf("DuracionH = %v, want %v", r.DuracionH, r.DistanciaKm/30)
			}
		})
	}
}

func TestFijo(t *testing.T) {
	t.Parallel()
	r, err := Fijo{Ruta: Ruta{DistanciaKm: 10, DuracionH: 0.5}}.Calcular(context.Background(), Coordenada{}, Coordenada{})
	if err != nil || r.DistanciaKm != 10 || r.DuracionH != 0.5 {
		t.Fatalf("got %v, %v", r, err)
	}
	if _, err := (Fijo{Err: ErrRutaNoDisponible}).Calcular(context.Background(), Coordenada{}, Coordenada{}); !errors.Is(err, ErrRutaNoDisponible) {
		t.Fatalf("err = %v", err)
	}
}
