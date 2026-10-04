package oferta

import (
	"sort"
	"time"
)

// Score de recomendación (RN-05, D-25): ordena las ofertas pendientes de una
// solicitud por precio, calificación y cumplimiento, sin cercanía.
// docs/ALGORITMO_SCORE.md.

const (
	pesoPrecio          = 0.50
	pesoCalificacion    = 0.3125
	pesoCumplimiento    = 0.1875
	calificacionNeutral = 3.5
)

// Puntuable reúne lo que necesita el score de una oferta.
type Puntuable struct {
	OfertaID string
	Precio   float64
	// Calificacion es nil si el Transportista no tiene reseñas: usa el valor
	// neutral 3,5 para no dejar últimos a los recién habilitados.
	Calificacion     *float64
	TasaCumplimiento float64 // 0..100
	CreadoEn         time.Time
}

// ordenarPorScore devuelve los índices de ofertas de mayor a menor score. Empata
// por calificación (con el neutral) y después por la oferta más antigua.
func ordenarPorScore(ofertas []Puntuable) []int {
	if len(ofertas) == 0 {
		return nil
	}
	minP, maxP := ofertas[0].Precio, ofertas[0].Precio
	for _, o := range ofertas[1:] {
		minP = min(minP, o.Precio)
		maxP = max(maxP, o.Precio)
	}
	scores := make([]float64, len(ofertas))
	califs := make([]float64, len(ofertas))
	for i, o := range ofertas {
		precioNorm := 1.0
		if maxP > minP {
			precioNorm = (maxP - o.Precio) / (maxP - minP)
		}
		calif := calificacionNeutral
		if o.Calificacion != nil {
			calif = *o.Calificacion
		}
		califs[i] = calif
		scores[i] = pesoPrecio*precioNorm +
			pesoCalificacion*((calif-1)/4) +
			pesoCumplimiento*(o.TasaCumplimiento/100)
	}
	orden := make([]int, len(ofertas))
	for i := range orden {
		orden[i] = i
	}
	sort.SliceStable(orden, func(a, b int) bool {
		i, j := orden[a], orden[b]
		if scores[i] != scores[j] {
			return scores[i] > scores[j]
		}
		if califs[i] != califs[j] {
			return califs[i] > califs[j]
		}
		return ofertas[i].CreadoEn.Before(ofertas[j].CreadoEn)
	})
	return orden
}
