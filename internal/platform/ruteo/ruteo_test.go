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
		key       string
		wantErr   bool
	}{
		{"aproximado en desarrollo", "aproximado", "development", "", false},
		{"aproximado en producción", "aproximado", "production", "", true},
		{"geoapify en producción", "geoapify", "production", "k", false},
		{"geoapify sin key", "geoapify", "development", "", true},
		{"google ya no es un proveedor", "google", "development", "k", true},
		{"desconocido", "otro", "development", "", true},
		{"fijo no se elige por configuración", "fijo", "development", "", true},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Nuevo(tc.proveedor, tc.entorno, tc.key)
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

func TestReducir(t *testing.T) {
	t.Parallel()
	puntos := make([]Coordenada, 1000)
	for i := range puntos {
		puntos[i] = Coordenada{Lat: float64(i)}
	}
	got := reducir(puntos, 300)
	if len(got) != 300 || got[0].Lat != 0 || got[299].Lat != 999 {
		t.Fatalf("reducir: %d puntos, primero %v, último %v", len(got), got[0], got[len(got)-1])
	}
	if len(reducir(puntos[:10], 300)) != 10 {
		t.Fatal("con pocos puntos no reduce")
	}
}

type contador struct {
	llamadas int
	err      error
}

func (c *contador) Calcular(context.Context, Coordenada, Coordenada) (Ruta, error) {
	c.llamadas++
	return Ruta{DistanciaKm: 1}, c.err
}

func TestConCache(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	a, b := Coordenada{1, 1}, Coordenada{2, 2}

	interno := &contador{}
	r := ConCache(interno)
	for range 3 {
		if _, err := r.Calcular(ctx, a, b); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.Calcular(ctx, b, a); err != nil {
		t.Fatal(err)
	}
	if interno.llamadas != 2 {
		t.Fatalf("llamadas = %d, want 2 (una por sentido)", interno.llamadas)
	}

	fallido := &contador{err: ErrRutaNoDisponible}
	r = ConCache(fallido)
	_, _ = r.Calcular(ctx, a, b)
	_, _ = r.Calcular(ctx, a, b)
	if fallido.llamadas != 2 {
		t.Fatalf("un error no se guarda: llamadas = %d", fallido.llamadas)
	}
}
