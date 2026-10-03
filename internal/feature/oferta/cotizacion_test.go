package oferta

import (
	"math"
	"testing"
)

// laboralSeed y operacionSeed son los valores de trabajo de la migración 0004.
var (
	laboralSeed = CostoLaboral{
		SalarioBasicoChofer: 1037544.76, SalarioBasicoAyudante: 963809.78,
		AdicionalesPctChofer: 18, AdicionalesPctAyudante: 16,
		ViaticosDiarios:            24724.33,
		ContribucionesSegSocialPct: 18, ObraSocialPct: 6, ArtPct: 10,
		SeguroVidaMensual: 424.62,
		HorasMensuales:    192, HorasDiarias: 8,
	}
	operacionSeed = Operacion{
		TiempoBaseOperacionMin: 10, TiempoEsperaMin: 30, TiempoPorObjetoMin: 2,
		TiempoPorKgMin: 0.01, TiempoPorM3Min: 8, TiempoPorMetroMin: 0.02,
		TiempoPorEscaleraMin: 5, EficienciaAyudante: 0.7,
	}
	// vehiculoEjemplo da 1200 $/h fijos y 210 $/km.
	vehiculoEjemplo = VehiculoCosto{
		CombustiblePrecioL: 1000, RendimientoKmL: 10,
		CantidadNeumaticos: 6, CostoNeumatico: 200000, VidaNeumaticoKm: 60000,
		CostoMantenimientoKm: 50,
		ValorCompra:          30000000, ValorResidual: 6000000, VidaUtilKm: 600000,
		SeguroMensual: 192000, PatenteMensual: 38400,
	}
)

func casiIgual(t *testing.T, nombre string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %v, want %v", nombre, got, want)
	}
}

func TestCostoPorHora(t *testing.T) {
	t.Parallel()
	casos := []struct {
		name string
		got  float64
		want float64
	}{
		// (remunerativo con SAC + cargas 34 % + viáticos de 24 días + seguro) / 192 h
		{"chofer con la seed", costoPorHoraChofer(laboralSeed), 12349.417338826388},
		{"ayudante con la seed", costoPorHoraAyudante(laboralSeed), 11545.833101743056},
		{"vehículo por hora: seguro y patente / horas", costoPorHoraVehiculo(vehiculoEjemplo, 192), 1200},
		{"vehículo por km: combustible + neumáticos + mantenimiento + depreciación", costoPorKmVehiculo(vehiculoEjemplo), 100 + 20 + 50 + 40},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			casiIgual(t, "costo", tc.got, tc.want, 1e-6)
		})
	}
}

func TestDuracionOperacion(t *testing.T) {
	t.Parallel()
	// 2 objetos de 50 kg y 1 m³: 10 + 30 + 2·2 + 100·0,01 + 2·8 = 61 min sin accesos.
	carga := []Item{{Objeto: Objeto{PesoKg: 50, LargoM: 1, AnchoM: 1, AltoM: 1}, Cantidad: 2}}
	casos := []struct {
		name            string
		origen, destino Acceso
		ayudantes       int
		wantMin         float64
	}{
		{"sin ayudantes ni accesos", Acceso{}, Acceso{}, 0, 61},
		{"un ayudante divide por 1,7", Acceso{}, Acceso{}, 1, 61 / 1.7},
		{"dos ayudantes divide por 1,98", Acceso{}, Acceso{}, 2, 61 / 1.98},
		{"tres ayudantes divide por 2,029", Acceso{}, Acceso{}, 3, 61 / (1 + 3*0.343)},
		{"pisos por escalera en origen y destino", Acceso{Pisos: 3}, Acceso{Pisos: 2}, 0, 61 + 5*5},
		{"con ascensor los pisos no cuentan", Acceso{Pisos: 3, AscensorUtilizable: true}, Acceso{Pisos: 2}, 0, 61 + 2*5},
		{"distancia a pie", Acceso{DistanciaVehiculoM: 30}, Acceso{DistanciaVehiculoM: 20}, 0, 61 + 50*0.02},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := duracionOperacion(operacionSeed, carga, tc.origen, tc.destino, tc.ayudantes)
			casiIgual(t, "horas", got, tc.wantMin/60, 1e-9)
		})
	}
}

func TestCadaAyudanteMejoraElTiempo(t *testing.T) {
	t.Parallel()
	carga := []Item{{Objeto: Objeto{PesoKg: 30, LargoM: 1, AnchoM: 1, AltoM: 1}, Cantidad: 5}}
	anterior := math.Inf(1)
	for n := 0; n <= maxAyudantes; n++ {
		h := duracionOperacion(operacionSeed, carga, Acceso{}, Acceso{}, n)
		if h >= anterior {
			t.Fatalf("con %d ayudantes %v h no mejora %v h", n, h, anterior)
		}
		anterior = h
	}
}

func TestPrecio(t *testing.T) {
	t.Parallel()
	casos := []struct {
		name                    string
		costo, margen, comision float64
		iva                     float64
		wantNeto, wantFinal     float64
	}{
		{"sin margen, comisión 15 %, IVA 21 %", 1000, 0, 15, 21, 1000 / 0.85, 1423.53},
		{"margen 10 %", 1000, 10, 15, 21, 1100 / 0.85, 1565.88},
		{"sin comisión ni IVA", 1000, 0, 0, 0, 1000, 1000},
		{"costo cero", 0, 10, 15, 21, 0, 0},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			neto := precioNeto(tc.costo, tc.margen, tc.comision)
			casiIgual(t, "neto", neto, tc.wantNeto, 1e-9)
			if got := precioFinal(neto, tc.iva); got != tc.wantFinal {
				t.Errorf("final = %v, want %v", got, tc.wantFinal)
			}
		})
	}
}

func TestRedondear2(t *testing.T) {
	t.Parallel()
	casos := []struct {
		name string
		in   float64
		want float64
	}{
		{"half-up con error de float", 1.005, 1.01},
		{"half-up 2,675", 2.675, 2.68},
		{"hacia abajo", 1.004, 1},
		{"exacto", 12.5, 12.5},
		{"monto grande", 1234567.895, 1234567.9},
		{"cero", 0, 0},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := redondear2(tc.in); got != tc.want {
				t.Errorf("redondear2(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestCotizar(t *testing.T) {
	t.Parallel()
	p := Parametros{Laboral: laboralSeed, Operacion: operacionSeed, MargenPct: 0, ComisionPct: 15, IvaPct: 21}
	ruta := Ruta{DistanciaKm: 10, DuracionH: 0.5}
	viaje := Viaje{Carga: []Item{{Objeto: Objeto{PesoKg: 50, LargoM: 1, AnchoM: 1, AltoM: 1}, Cantidad: 2}}}
	opH := 61.0 / 60 // ver TestDuracionOperacion

	casos := []struct {
		name        string
		viajes      []Viaje
		ayudantes   int
		adicionales float64
	}{
		{"un viaje sin ayudantes", []Viaje{viaje}, 0, 0},
		{"dos viajes iguales", []Viaje{viaje, viaje}, 0, 0},
		{"un viaje con un ayudante y costos adicionales", []Viaje{viaje}, 1, 5000},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := cotizar(p, vehiculoEjemplo, tc.viajes, ruta, Acceso{}, Acceso{}, tc.ayudantes, tc.adicionales)

			n := float64(len(tc.viajes))
			op := opH / (1 + math.Pow(0.7, float64(tc.ayudantes))*float64(tc.ayudantes))
			h := 0.5*2 + op // ida y vuelta + operación
			laborH := 12349.417338826388 + float64(tc.ayudantes)*11545.833101743056
			wantLaboral := n * laborH * h
			wantVehiculo := n * (1200*h + 210*20)
			wantOperativo := wantLaboral + wantVehiculo + tc.adicionales

			casiIgual(t, "DuracionOperacionH", d.DuracionOperacionH, n*op, 1e-9)
			casiIgual(t, "CostoLaboral", d.CostoLaboral, wantLaboral, 1e-6)
			casiIgual(t, "CostoVehiculo", d.CostoVehiculo, wantVehiculo, 1e-6)
			casiIgual(t, "CostoOperativo", d.CostoOperativo, wantOperativo, 1e-6)
			casiIgual(t, "PrecioNeto", d.PrecioNeto, wantOperativo/0.85, 1e-6)
			if want := redondear2(wantOperativo / 0.85 * 1.21); d.PrecioFinal != want {
				t.Errorf("PrecioFinal = %v, want %v", d.PrecioFinal, want)
			}
			if d.DistanciaKm != 10 || d.DuracionRutaH != 0.5 || d.ComisionPct != 15 || d.IvaPct != 21 {
				t.Errorf("parámetros no copiados: %+v", d)
			}
		})
	}
}
