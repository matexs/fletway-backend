package oferta

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/bavix/boxpacker3/v2"
)

// Planificación de viajes (RN-02): cuántos viajes hacen falta para llevar la
// carga en un vehículo real. docs/ALGORITMO_VIAJES_EMPAQUETADO.md §3–§5. Es una
// estimación para el precio (greedy de una pasada), no el mínimo de viajes.

// ErrNoFactible indica que la carga no se puede llevar con el vehículo. El
// *NoFactibleError que lo envuelve trae el motivo por objeto.
var ErrNoFactible = errors.New("carga no factible con este vehículo")

const (
	// maxViajes es la salvaguarda: con más viajes que esto no se oferta con ese
	// vehículo.
	maxViajes = 20
	// minApoyoBase es la fracción de la base de cada objeto que tiene que estar
	// apoyada (fijo en el código, D-24).
	minApoyoBase = 0.6
)

// Objeto es una fila de solicitud_objeto: peso y medidas por unidad. Alto es el
// eje vertical.
type Objeto struct {
	Nombre                string
	PesoKg                float64
	LargoM, AnchoM, AltoM float64
	RotacionHorizontal    bool
	RotacionVertical      bool
	Apilable              bool
}

func (o Objeto) volumenM3() float64 { return o.LargoM * o.AnchoM * o.AltoM }

// Item es un objeto con su cantidad.
type Item struct {
	Objeto   Objeto
	Cantidad int
}

// Vehiculo es la capacidad útil del vehículo con el que se oferta.
type Vehiculo struct {
	Patente                           string
	PesoUtilKg                        float64 // vehiculo.peso_maximo_kg, ya es carga útil
	LargoUtilM, AnchoUtilM, AltoUtilM float64
}

// Viaje es un viaje físico dentro de la oferta, con su parte de la carga (no la
// tabla viaje).
type Viaje struct {
	Carga []Item
}

// Plan es el resultado de planificarViajes.
type Plan struct {
	Viajes []Viaje
	// CotaInferior es la cantidad mínima teórica de viajes según la librería;
	// sirve para evaluar la calidad de la estimación.
	CotaInferior int
	// RotacionNoAplicada son los objetos con rotación horizontal prohibida y
	// vertical permitida: la librería no tiene esa restricción y se ignora (§2.1).
	RotacionNoAplicada []string
}

// NoFactibleError explica por qué la carga no entra. Envuelve ErrNoFactible.
type NoFactibleError struct {
	Motivos []string
}

func (e *NoFactibleError) Error() string {
	return fmt.Sprintf("%v: %v", ErrNoFactible, e.Motivos)
}

func (e *NoFactibleError) Unwrap() error { return ErrNoFactible }

// planificarViajes reparte la carga en viajes del vehículo. Devuelve un
// *NoFactibleError si algún objeto no entra o si hacen falta más de maxViajes, y
// el error de ctx si vence (el llamador pone el límite de 5 s, D-24).
func planificarViajes(ctx context.Context, v Vehiculo, carga []Item) (Plan, error) {
	// Cota rápida por peso y volumen: si ni idealmente entra en maxViajes, se corta
	// antes de llamar a la librería, que tarda en descubrirlo.
	var pesoTotal, volumenTotal float64
	for _, it := range carga {
		pesoTotal += it.Objeto.PesoKg * float64(it.Cantidad)
		volumenTotal += it.Objeto.volumenM3() * float64(it.Cantidad)
	}
	volumenUtil := v.LargoUtilM * v.AnchoUtilM * v.AltoUtilM
	if math.Ceil(pesoTotal/v.PesoUtilKg) > maxViajes || math.Ceil(volumenTotal/volumenUtil) > maxViajes {
		return Plan{}, &NoFactibleError{Motivos: []string{motivoMasViajes}}
	}

	var plan Plan
	items := make([]*boxpacker3.Item, 0, len(carga))
	for idx, it := range carga {
		rot, ejes := rotacionDe(it.Objeto)
		if it.Objeto.RotacionVertical && !it.Objeto.RotacionHorizontal {
			plan.RotacionNoAplicada = append(plan.RotacionNoAplicada, it.Objeto.Nombre)
		}
		item, err := boxpacker3.NewItemFromSpec(boxpacker3.ItemSpec{
			ID:           strconv.Itoa(idx), // índice en carga, para reagrupar después
			Width:        it.Objeto.LargoM,
			Height:       it.Objeto.AnchoM,
			Depth:        it.Objeto.AltoM, // eje vertical
			Weight:       it.Objeto.PesoKg,
			Quantity:     it.Cantidad,
			Rotation:     rot,
			VerticalAxes: ejes,
			NothingOnTop: !it.Objeto.Apilable,
		})
		if err != nil {
			return Plan{}, fmt.Errorf("ítem %q: %w", it.Objeto.Nombre, err)
		}
		items = append(items, item)
	}

	caja, err := boxpacker3.NewBoxFromSpec(boxpacker3.BoxSpec{
		ID:          v.Patente,
		OuterWidth:  v.LargoUtilM,
		OuterHeight: v.AnchoUtilM,
		OuterDepth:  v.AltoUtilM, // eje vertical
		MaxWeight:   v.PesoUtilKg,
		Quantity:    maxViajes,
	})
	if err != nil {
		return Plan{}, fmt.Errorf("vehículo %s: %w", v.Patente, err)
	}

	packer := boxpacker3.NewPacker(
		boxpacker3.WithAlgorithm(boxpacker3.NewGreedy(boxpacker3.OrderDecreasing, boxpacker3.SelectFirstFit)),
		boxpacker3.WithFinishers(),
		boxpacker3.WithRules(boxpacker3.Rules{MinSupportRatio: minApoyoBase}),
	)
	result, err := packer.Pack(ctx, []*boxpacker3.Box{caja}, items)
	if err != nil {
		return Plan{}, fmt.Errorf("empaquetar carga: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return Plan{}, fmt.Errorf("empaquetar carga: %w", err)
	}
	if len(result.Unpacked) > 0 {
		return Plan{}, noFactible(carga, result.Unpacked)
	}

	plan.Viajes = dividirEnViajes(carga, result)
	plan.CotaInferior = result.Report.Bound.Boxes
	return plan, nil
}

// rotacionDe traduce los flags del objeto a la librería (§2.1).
func rotacionDe(o Objeto) (boxpacker3.Rotation, []boxpacker3.Axis) {
	switch {
	case !o.RotacionVertical && !o.RotacionHorizontal:
		return boxpacker3.RotationNever, nil
	case !o.RotacionVertical:
		// Queda parado: el alto sigue siendo vertical y puede girar sobre el piso.
		return boxpacker3.RotationBestFit, []boxpacker3.Axis{boxpacker3.DepthAxis}
	default:
		return boxpacker3.RotationBestFit, nil
	}
}

func dividirEnViajes(carga []Item, result *boxpacker3.Result) []Viaje {
	viajes := make([]Viaje, 0, len(result.Boxes))
	for _, box := range result.Boxes {
		if len(box.Items) == 0 {
			continue
		}
		porFila := map[int]int{}
		for _, p := range box.Items {
			idx, _ := strconv.Atoi(p.Item.ID())
			porFila[idx]++
		}
		var v Viaje
		for idx := range carga {
			if n := porFila[idx]; n > 0 {
				v.Carga = append(v.Carga, Item{Objeto: carga[idx].Objeto, Cantidad: n})
			}
		}
		viajes = append(viajes, v)
	}
	return viajes
}

var motivoMasViajes = fmt.Sprintf("la carga necesita más de %d viajes con este vehículo", maxViajes)

// motivos traduce los motivos de la librería para el Transportista.
var motivos = map[boxpacker3.Reason]string{
	boxpacker3.ReasonTooBig:            "no entra en el vehículo en ninguna posición",
	boxpacker3.ReasonTooHeavy:          "supera la carga útil del vehículo",
	boxpacker3.ReasonNoOrientationFits: "no entra en la posición en que tiene que viajar",
}

func noFactible(carga []Item, unpacked []boxpacker3.UnpackedItem) error {
	e := &NoFactibleError{}
	vistos := map[string]bool{}
	for _, u := range unpacked {
		idx, _ := strconv.Atoi(u.Item.ID())
		m, ok := motivos[u.Reason]
		switch {
		case u.Reason == boxpacker3.ReasonNoRoom:
			m = motivoMasViajes
		case ok:
			m = carga[idx].Objeto.Nombre + ": " + m
		default:
			m = carga[idx].Objeto.Nombre + ": no se pudo ubicar en el vehículo"
		}
		if !vistos[m] {
			vistos[m] = true
			e.Motivos = append(e.Motivos, m)
		}
	}
	return e
}
