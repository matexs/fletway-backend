package oferta

import (
	"math"
	"strconv"
)

// Funciones puras del precio de una oferta (RN-01). Fórmulas y nombres de
// docs/ALGORITMO_COTIZACION.md §4 y §5, con los ajustes de D-24. Los porcentajes
// llegan en 0–100, como en la base. No hay redondeo intermedio: sólo el precio
// final se redondea (D-11).

// CostoLaboral es la fila vigente de config_costo_laboral.
type CostoLaboral struct {
	SalarioBasicoChofer, SalarioBasicoAyudante        float64
	AdicionalesPctChofer, AdicionalesPctAyudante      float64
	ViaticosDiarios                                   float64
	ContribucionesSegSocialPct, ObraSocialPct, ArtPct float64
	SeguroVidaMensual                                 float64
	HorasMensuales, HorasDiarias                      float64
}

// Operacion es la fila vigente de config_operacion (tiempos de carga y
// descarga, en minutos).
type Operacion struct {
	TiempoBaseOperacionMin float64
	TiempoEsperaMin        float64
	TiempoPorObjetoMin     float64
	TiempoPorKgMin         float64
	TiempoPorM3Min         float64
	TiempoPorMetroMin      float64
	TiempoPorEscaleraMin   float64
	EficienciaAyudante     float64
}

// Parametros son los valores de plataforma vigentes al cotizar.
type Parametros struct {
	Laboral     CostoLaboral
	Operacion   Operacion
	MargenPct   float64 // config_margen (D-15)
	ComisionPct float64 // config_comision
	IvaPct      float64 // config_impuesto
}

// VehiculoCosto son los costos de referencia del tipo de vehículo
// (config_costo_vehiculo vigente, D-34): los define la plataforma, no el
// Transportista.
type VehiculoCosto struct {
	CombustiblePrecioL   float64
	RendimientoKmL       float64
	CantidadNeumaticos   int
	CostoNeumatico       float64
	VidaNeumaticoKm      float64
	CostoMantenimientoKm float64
	ValorCompra          float64
	ValorResidual        float64
	VidaUtilKm           float64
	SeguroMensual        float64
	PatenteMensual       float64
}

// Acceso describe cómo se llega a la carga en origen o destino.
type Acceso struct {
	Pisos              int
	AscensorUtilizable bool
	DistanciaVehiculoM float64
}

// Ruta es el trayecto de ida origen → destino.
type Ruta struct {
	DistanciaKm float64
	DuracionH   float64
}

// Desglose es el resultado de cotizar una oferta; se guarda en oferta_costo y el
// precio final en oferta.precio_calculado (ALGORITMO_COTIZACION.md §6).
type Desglose struct {
	DistanciaKm, DuracionRutaH, DuracionOperacionH float64
	CostoLaboral, CostoVehiculo, CostosAdicionales float64
	CostoOperativo                                 float64
	MargenPct, ComisionPct, IvaPct                 float64
	PrecioNeto                                     float64
	PrecioFinal                                    float64 // redondeado a 2 decimales
}

// aguinaldoFactor es el sueldo anual complementario: 13 sueldos en 12 meses.
const aguinaldoFactor = 13.0 / 12.0

func costoPorHoraTrabajo(c CostoLaboral, salarioBasico, adicionalesPct float64) float64 {
	remunerativo := salarioBasico * (1 + adicionalesPct/100) * aguinaldoFactor
	cargas := remunerativo * (c.ContribucionesSegSocialPct + c.ObraSocialPct + c.ArtPct) / 100
	diasMensuales := c.HorasMensuales / c.HorasDiarias
	noRemunerativo := c.ViaticosDiarios*diasMensuales + c.SeguroVidaMensual
	return (remunerativo + cargas + noRemunerativo) / c.HorasMensuales
}

func costoPorHoraChofer(c CostoLaboral) float64 {
	return costoPorHoraTrabajo(c, c.SalarioBasicoChofer, c.AdicionalesPctChofer)
}

func costoPorHoraAyudante(c CostoLaboral) float64 {
	return costoPorHoraTrabajo(c, c.SalarioBasicoAyudante, c.AdicionalesPctAyudante)
}

func costoPorHoraVehiculo(v VehiculoCosto, horasMensuales float64) float64 {
	return (v.SeguroMensual + v.PatenteMensual) / horasMensuales
}

func costoPorKmVehiculo(v VehiculoCosto) float64 {
	combustible := v.CombustiblePrecioL / v.RendimientoKmL
	neumaticos := float64(v.CantidadNeumaticos) * v.CostoNeumatico / v.VidaNeumaticoKm
	depreciacion := (v.ValorCompra - v.ValorResidual) / v.VidaUtilKm
	return combustible + neumaticos + v.CostoMantenimientoKm + depreciacion
}

// duracionOperacion es el tiempo de carga y descarga de un viaje, en horas
// (§4.3). La espera se suma una vez por viaje (D-24). Los ayudantes dividen el
// trabajo por 1 + e^n·n.
func duracionOperacion(op Operacion, carga []Item, origen, destino Acceso, ayudantes int) float64 {
	var nObjetos int
	var pesoKg, volumenM3 float64
	for _, it := range carga {
		nObjetos += it.Cantidad
		pesoKg += it.Objeto.PesoKg * float64(it.Cantidad)
		volumenM3 += it.Objeto.volumenM3() * float64(it.Cantidad)
	}
	distanciaM := origen.DistanciaVehiculoM + destino.DistanciaVehiculoM
	pisos := 0
	if !origen.AscensorUtilizable {
		pisos += origen.Pisos
	}
	if !destino.AscensorUtilizable {
		pisos += destino.Pisos
	}

	trabajoMin := op.TiempoBaseOperacionMin +
		op.TiempoEsperaMin +
		float64(nObjetos)*op.TiempoPorObjetoMin +
		pesoKg*op.TiempoPorKgMin +
		volumenM3*op.TiempoPorM3Min +
		distanciaM*op.TiempoPorMetroMin +
		float64(pisos)*op.TiempoPorEscaleraMin

	n := float64(ayudantes)
	eficiencia := 1 + math.Pow(op.EficienciaAyudante, n)*n
	return trabajoMin / eficiencia / 60
}

// cotizar calcula el costo operativo de todos los viajes y el precio al Cliente
// (§4.4 y §5). Cada viaje va y vuelve: se cobran el doble de horas y de km de la
// ruta (D-24). costosAdicionales se suma una vez por oferta.
func cotizar(p Parametros, vc VehiculoCosto, viajes []Viaje, ruta Ruta, origen, destino Acceso, ayudantes int, costosAdicionales float64) Desglose {
	d := Desglose{
		DistanciaKm: ruta.DistanciaKm, DuracionRutaH: ruta.DuracionH, CostosAdicionales: costosAdicionales,
		MargenPct: p.MargenPct, ComisionPct: p.ComisionPct, IvaPct: p.IvaPct,
	}
	costoLaborH := costoPorHoraChofer(p.Laboral) + float64(ayudantes)*costoPorHoraAyudante(p.Laboral)
	costoVehiculoH := costoPorHoraVehiculo(vc, p.Laboral.HorasMensuales)
	costoVehiculoKm := costoPorKmVehiculo(vc)
	for _, v := range viajes {
		operacionH := duracionOperacion(p.Operacion, v.Carga, origen, destino, ayudantes)
		h := ruta.DuracionH*2 + operacionH
		km := ruta.DistanciaKm * 2
		d.DuracionOperacionH += operacionH
		d.CostoLaboral += costoLaborH * h
		d.CostoVehiculo += costoVehiculoH*h + costoVehiculoKm*km
	}
	d.CostoOperativo = d.CostoLaboral + d.CostoVehiculo + d.CostosAdicionales
	d.PrecioNeto = precioNeto(d.CostoOperativo, p.MargenPct, p.ComisionPct)
	d.PrecioFinal = precioFinal(d.PrecioNeto, p.IvaPct)
	return d
}

// precioNeto es lo que paga el Cliente antes de impuestos: la comisión se
// calcula sobre este valor, no sobre el costo.
func precioNeto(costoOperativo, margenPct, comisionPct float64) float64 {
	return costoOperativo * (1 + margenPct/100) / (1 - comisionPct/100)
}

func precioFinal(neto, ivaPct float64) float64 {
	return redondear2(neto * (1 + ivaPct/100))
}

// redondear2 redondea half-up a 2 decimales. Antes se recorta a 6 decimales el
// valor en centavos para que el error de float64 (1.005*100 = 100.4999...) no
// cambie el redondeo.
func redondear2(x float64) float64 {
	centavos, _ := strconv.ParseFloat(strconv.FormatFloat(x*100, 'f', 6, 64), 64)
	return math.Floor(centavos+0.5) / 100
}
