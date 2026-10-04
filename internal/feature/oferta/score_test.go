package oferta

import (
	"testing"
	"time"
)

func calif(v float64) *float64 { return &v }

func TestOrdenarPorScore(t *testing.T) {
	t.Parallel()
	t0 := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	casos := []struct {
		name    string
		ofertas []Puntuable
		want    []string
	}{
		{
			name: "a igual reputación gana la más barata",
			ofertas: []Puntuable{
				{OfertaID: "cara", Precio: 120000, Calificacion: calif(4), TasaCumplimiento: 100, CreadoEn: t0},
				{OfertaID: "barata", Precio: 90000, Calificacion: calif(4), TasaCumplimiento: 100, CreadoEn: t0},
				{OfertaID: "media", Precio: 100000, Calificacion: calif(4), TasaCumplimiento: 100, CreadoEn: t0},
			},
			want: []string{"barata", "media", "cara"},
		},
		{
			name: "mismo precio: decide la reputación",
			ofertas: []Puntuable{
				{OfertaID: "regular", Precio: 100000, Calificacion: calif(3), TasaCumplimiento: 80, CreadoEn: t0},
				{OfertaID: "excelente", Precio: 100000, Calificacion: calif(5), TasaCumplimiento: 100, CreadoEn: t0},
			},
			want: []string{"excelente", "regular"},
		},
		{
			name: "sin reseñas usa 3,5 y no queda último",
			ofertas: []Puntuable{
				{OfertaID: "mala", Precio: 100000, Calificacion: calif(2), TasaCumplimiento: 100, CreadoEn: t0},
				{OfertaID: "nuevo", Precio: 100000, Calificacion: nil, TasaCumplimiento: 100, CreadoEn: t0},
				{OfertaID: "buena", Precio: 100000, Calificacion: calif(4.5), TasaCumplimiento: 100, CreadoEn: t0},
			},
			want: []string{"buena", "nuevo", "mala"},
		},
		{
			name: "empate exacto: decide la más antigua",
			ofertas: []Puntuable{
				{OfertaID: "nueva", Precio: 100000, Calificacion: calif(4), TasaCumplimiento: 100, CreadoEn: t0.Add(time.Hour)},
				{OfertaID: "vieja", Precio: 100000, Calificacion: calif(4), TasaCumplimiento: 100, CreadoEn: t0},
			},
			want: []string{"vieja", "nueva"},
		},
		{
			// Mismo precio; 4,5 sin cumplimiento suma 0,2734375 y 3 con 62,5 % de
			// cumplimiento también (valores exactos en binario): gana la mejor
			// calificación aunque sea la oferta más nueva.
			name: "empate de score: desempata la calificación",
			ofertas: []Puntuable{
				{OfertaID: "tres", Precio: 100000, Calificacion: calif(3), TasaCumplimiento: 62.5, CreadoEn: t0},
				{OfertaID: "cuatroymedio", Precio: 100000, Calificacion: calif(4.5), TasaCumplimiento: 0, CreadoEn: t0.Add(time.Hour)},
			},
			want: []string{"cuatroymedio", "tres"},
		},
		{
			name:    "una sola oferta: sin división por cero",
			ofertas: []Puntuable{{OfertaID: "unica", Precio: 50000, TasaCumplimiento: 100, CreadoEn: t0}},
			want:    []string{"unica"},
		},
		{
			name: "el precio pesa más que una estrella",
			ofertas: []Puntuable{
				{OfertaID: "cara5", Precio: 150000, Calificacion: calif(5), TasaCumplimiento: 100, CreadoEn: t0},
				{OfertaID: "barata4", Precio: 100000, Calificacion: calif(4), TasaCumplimiento: 100, CreadoEn: t0},
			},
			want: []string{"barata4", "cara5"},
		},
		{name: "sin ofertas", ofertas: nil, want: nil},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			orden := ordenarPorScore(tc.ofertas)
			if len(orden) != len(tc.want) {
				t.Fatalf("orden = %v, want %v", orden, tc.want)
			}
			for pos, idx := range orden {
				if got := tc.ofertas[idx].OfertaID; got != tc.want[pos] {
					t.Fatalf("posición %d = %s, want %s (orden %v)", pos, got, tc.want[pos], orden)
				}
			}
		})
	}
}
