# Fletway — Algoritmo del score de recomendación

**Estado:** definido para implementar (D-25, 2026-10-01). Corresponde a **RN-05** (top 3 por score) y a **RF-07** (el Cliente elige entre las ofertas).

**Enmienda a la ERS:** RN-05 combina "precio, calificación, cercanía y tasa de cumplimiento". La **cercanía se elimina** del score (D-25): el Transportista no tiene ubicación guardada y ya se decidió no calcular su distancia al origen (`ALGORITMO_COTIZACION.md` §0.2). El score usa los otros tres factores.

---

## 1. Entradas

Para cada oferta `pendiente` de una solicitud:

| Factor | Columna | Escala |
|---|---|---|
| Precio | `oferta.precio_calculado` | pesos |
| Calificación | `transportista.calificacion_promedio` | 1 a 5, o `NULL` si no tiene reseñas |
| Cumplimiento | `transportista.tasa_cumplimiento` | 0 a 100 (inicial 100, D-29) |

---

## 2. Fórmula

```
score = 0.50 * precio_norm + 0.3125 * calificacion_norm + 0.1875 * cumplimiento_norm
```

Los pesos salen de la propuesta original (precio 0,40, calificación 0,25, cercanía 0,20, cumplimiento 0,15) **renormalizados** al quitar la cercanía: cada peso se divide por 0,80, así que se mantienen las proporciones entre los tres factores y la suma sigue siendo 1.

Normalización de cada factor a 0..1 (más alto = mejor):

- **`precio_norm`:** min-max **dentro de las ofertas de esa misma solicitud**, invertido (el precio más bajo vale 1). Si todas las ofertas tienen el mismo precio, `precio_norm = 1` para todas.
  `precio_norm = (max - precio) / (max - min)`
- **`calificacion_norm`:** `(calificacion - 1) / 4`. Un Transportista **sin reseñas** usa un valor neutral de **3,5** (no 0), para no dejar siempre últimos a los recién habilitados.
- **`cumplimiento_norm`:** `tasa_cumplimiento / 100`.

**Desempate:** calificación descendente (con el 3,5 neutral para quien no tiene reseñas) y después la oferta más antigua (`oferta.creado_en` ascendente).

---

## 3. Pseudocódigo (Go)

```go
const (
    pesoPrecio       = 0.50
    pesoCalificacion = 0.3125
    pesoCumplimiento = 0.1875
    calificacionNeutral = 3.5
)

// OfertaPuntuable reúne lo que necesita el score de una oferta.
type OfertaPuntuable struct {
    OfertaID          string
    Precio            float64   // oferta.precio_calculado
    Calificacion      *float64  // transportista.calificacion_promedio (nil = sin reseñas)
    TasaCumplimiento  float64   // transportista.tasa_cumplimiento (0..100)
    CreadoEn          time.Time // oferta.creado_en
}

// ordenarPorScore devuelve las ofertas ordenadas de mayor a menor score.
// Función pura: sin I/O, testeable sin base (CLAUDE.md §5).
func ordenarPorScore(ofertas []OfertaPuntuable) []OfertaPuntuable {
    min, max := rangoPrecios(ofertas)
    type puntuada struct {
        o     OfertaPuntuable
        score float64
        calif float64
    }
    ps := make([]puntuada, 0, len(ofertas))
    for _, o := range ofertas {
        precioNorm := 1.0
        if max > min {
            precioNorm = (max - o.Precio) / (max - min)
        }
        calif := calificacionNeutral
        if o.Calificacion != nil {
            calif = *o.Calificacion
        }
        score := pesoPrecio*precioNorm +
            pesoCalificacion*((calif-1)/4) +
            pesoCumplimiento*(o.TasaCumplimiento/100)
        ps = append(ps, puntuada{o: o, score: score, calif: calif})
    }
    sort.SliceStable(ps, func(i, j int) bool {
        if ps[i].score != ps[j].score {
            return ps[i].score > ps[j].score
        }
        if ps[i].calif != ps[j].calif {
            return ps[i].calif > ps[j].calif
        }
        return ps[i].o.CreadoEn.Before(ps[j].o.CreadoEn)
    })
    out := make([]OfertaPuntuable, len(ps))
    for i, p := range ps {
        out[i] = p.o
    }
    return out
}
```

---

## 4. Uso en la API

- `GET /api/solicitudes/{id}/ofertas` devuelve las **3** primeras según el score.
- "Ver más" (`?ver_mas=true` con `?limit=&cursor=`) continúa la **misma lista ordenada** después de la posición 3.
- El score se calcula al leer (las ofertas de una solicitud son pocas); no se persiste.
- Al Cliente se le exponen sólo los datos de D-23/R-05 (`docs/PLAN_CONSTRUCCION.md`): nunca el desglose de costo.

---

## 5. Tests obligatorios (table-driven)

- Ofertas con precios distintos: la más barata gana a igualdad de reputación.
- Todas con el mismo precio: decide la reputación.
- Transportista sin reseñas: usa 3,5, no queda último por eso.
- Empate exacto: desempata calificación y después antigüedad.
- Una sola oferta: `precio_norm = 1`, sin división por cero.
