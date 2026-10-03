package oferta

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// furgon es un utilitario de 3 × 1,7 × 1,7 m y 1500 kg de carga útil.
var furgon = Vehiculo{Patente: "AB123CD", PesoUtilKg: 1500, LargoUtilM: 3, AnchoUtilM: 1.7, AltoUtilM: 1.7}

func caja(nombre string, peso, l, a, h float64) Objeto {
	return Objeto{Nombre: nombre, PesoKg: peso, LargoM: l, AnchoM: a, AltoM: h,
		RotacionHorizontal: true, RotacionVertical: true, Apilable: true}
}

func totalPorNombre(viajes []Viaje) map[string]int {
	out := map[string]int{}
	for _, v := range viajes {
		for _, it := range v.Carga {
			out[it.Objeto.Nombre] += it.Cantidad
		}
	}
	return out
}

func TestPlanificarViajes(t *testing.T) {
	t.Parallel()
	ropero := Objeto{Nombre: "Ropero", PesoKg: 80, LargoM: 1.2, AnchoM: 0.6, AltoM: 1.9,
		RotacionHorizontal: true, RotacionVertical: false, Apilable: false}

	casos := []struct {
		name       string
		carga      []Item
		wantViajes int
		wantMotivo string // si no está vacío, se espera ErrNoFactible con este texto
	}{
		{
			name:       "entra en un viaje",
			carga:      []Item{{caja("Caja", 10, 0.5, 0.4, 0.4), 20}, {caja("Heladera", 70, 0.7, 0.7, 1.6), 1}},
			wantViajes: 1,
		},
		{
			name:       "el peso obliga a dos viajes",
			carga:      []Item{{caja("Bolsa de cemento", 50, 0.6, 0.4, 0.15), 40}},
			wantViajes: 2,
		},
		{
			name:       "no entra en ninguna posición",
			carga:      []Item{{caja("Contenedor", 100, 4, 2, 2), 1}},
			wantMotivo: "Contenedor: no entra en el vehículo en ninguna posición",
		},
		{
			name:       "un objeto solo supera la carga útil",
			carga:      []Item{{caja("Caja fuerte", 1600, 1, 1, 1), 1}},
			wantMotivo: "Caja fuerte: supera la carga útil del vehículo",
		},
		{
			name:       "entraría acostado pero viaja parado",
			carga:      []Item{{ropero, 1}},
			wantMotivo: "Ropero: no entra en la posición en que tiene que viajar",
		},
		{
			name:       "cota rápida: más de 20 viajes por peso",
			carga:      []Item{{caja("Ladrillos", 1000, 0.5, 0.5, 0.5), 31}},
			wantMotivo: "la carga necesita más de 20 viajes",
		},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			plan, err := planificarViajes(context.Background(), furgon, tc.carga)
			if tc.wantMotivo != "" {
				var nf *NoFactibleError
				if !errors.Is(err, ErrNoFactible) || !errors.As(err, &nf) {
					t.Fatalf("err = %v, want ErrNoFactible", err)
				}
				if !strings.Contains(strings.Join(nf.Motivos, "; "), tc.wantMotivo) {
					t.Fatalf("motivos = %v, want %q", nf.Motivos, tc.wantMotivo)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.Viajes) != tc.wantViajes {
				t.Fatalf("viajes = %d, want %d", len(plan.Viajes), tc.wantViajes)
			}
			if plan.CotaInferior < 1 || plan.CotaInferior > len(plan.Viajes) {
				t.Errorf("CotaInferior = %d con %d viajes", plan.CotaInferior, len(plan.Viajes))
			}
			// Toda la carga aparece repartida en los viajes, sin perder ni duplicar.
			got := totalPorNombre(plan.Viajes)
			for _, it := range tc.carga {
				if got[it.Objeto.Nombre] != it.Cantidad {
					t.Errorf("%s: %d en los viajes, want %d", it.Objeto.Nombre, got[it.Objeto.Nombre], it.Cantidad)
				}
			}
		})
	}
}

func TestPlanificarViajesRotacionInformativa(t *testing.T) {
	t.Parallel()
	o := caja("Espejo", 10, 1, 0.1, 0.8)
	o.RotacionHorizontal = false
	plan, err := planificarViajes(context.Background(), furgon, []Item{{o, 1}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.RotacionNoAplicada) != 1 || plan.RotacionNoAplicada[0] != "Espejo" {
		t.Fatalf("RotacionNoAplicada = %v", plan.RotacionNoAplicada)
	}
}

func TestPlanificarViajesContextoVencido(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := planificarViajes(ctx, furgon, []Item{{caja("Caja", 10, 0.5, 0.4, 0.4), 20}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
