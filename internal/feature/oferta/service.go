package oferta

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/decimal"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
	"github.com/matexs/fletway-backend/internal/platform/ruteo"
)

// horaArgentina es la zona horaria del negocio (UTC-3, sin horario de verano).
var horaArgentina = time.FixedZone("ART", -3*60*60)

// limiteCalculo es el tiempo máximo del cálculo de viajes (D-24).
const limiteCalculo = 5 * time.Second

var (
	errNoEsTransportista = httpx.Forbidden("no_es_transportista",
		"esta acción es sólo para Transportistas registrados")
	errNoHabilitado = httpx.Forbidden("transportista_no_habilitado",
		"tu cuenta todavía no está habilitada para ofertar")
	errNoDisponible = httpx.Conflict("transportista_no_disponible",
		"activá tu disponibilidad para ofertar")
	errSolicitudNoDisponible = httpx.NotFound("solicitud_no_disponible",
		"la solicitud no existe o ya no está disponible para ofertar")
	errVehiculoAPI         = httpx.NotFound("vehiculo_no_encontrado", "no existe el vehículo o no es tuyo")
	errVehiculoInactivoAPI = httpx.Conflict("vehiculo_inactivo", "el vehículo está inactivo")
	errDuplicadaAPI        = httpx.Conflict("oferta_duplicada",
		"ya tenés una oferta vigente con ese vehículo para esta solicitud")
	errRutaAPI = httpx.BadRequest("ruta_no_disponible",
		"no pudimos calcular el recorrido; probá de nuevo más tarde")
	errCalculoDemoradoAPI = httpx.BadRequest("calculo_demorado",
		"el cálculo de viajes tardó demasiado; probá con un vehículo más grande")
	errNoEncontradaAPI = httpx.NotFound("oferta_no_encontrada", "no existe la oferta o no es tuya")
	errNoRetirableAPI  = httpx.Conflict("oferta_no_retirable", "sólo se retira una oferta pendiente")
)

// Service calcula, crea, retira y lista las ofertas del Transportista (RF-17,
// RN-01, RN-02).
type Service struct {
	repo  *repository
	ruta  ruteo.Ruteador
	ahora func() time.Time
}

// NewService crea el Service. ruta da la distancia y la duración del trayecto
// (D-24).
func NewService(db *database.DB, ruta ruteo.Ruteador) *Service {
	return &Service{repo: &repository{db: db}, ruta: ruta, ahora: time.Now}
}

func (s *Service) hoy() time.Time {
	a := s.ahora().In(horaArgentina)
	return time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
}

// calculo es una oferta calculada y todavía no guardada.
type calculo struct {
	Viajes   int
	Desglose Desglose
}

// calcular arma la oferta: valida al Transportista, la solicitud y el vehículo,
// planifica los viajes (RN-02) y calcula el precio (RN-01). No escribe en la base.
func (s *Service) calcular(ctx context.Context, id database.Identity, solicitudID string, req OfertaRequest) (calculo, error) {
	p, err := s.repo.perfil(ctx, id)
	switch {
	case err != nil:
		return calculo{}, err
	case !p.Existe:
		return calculo{}, errNoEsTransportista
	case !p.Habilitado:
		return calculo{}, errNoHabilitado
	case !p.Disponible:
		return calculo{}, errNoDisponible
	}
	compatible, err := s.repo.compatible(ctx, id, solicitudID)
	if err != nil {
		return calculo{}, err
	}
	if !compatible {
		return calculo{}, errSolicitudNoDisponible
	}

	e, err := s.repo.entrada(ctx, id, solicitudID, req.VehiculoID)
	switch {
	case errors.Is(err, errVehiculoNoEncontrado):
		return calculo{}, errVehiculoAPI
	case errors.Is(err, errVehiculoInactivo):
		return calculo{}, errVehiculoInactivoAPI
	case errors.Is(err, errSolicitudInexistente):
		return calculo{}, errSolicitudNoDisponible
	case err != nil:
		return calculo{}, err
	}

	planCtx, cancel := context.WithTimeout(ctx, limiteCalculo)
	defer cancel()
	plan, err := planificarViajes(planCtx, e.Vehiculo, e.Carga)
	var nf *NoFactibleError
	switch {
	case errors.As(err, &nf):
		return calculo{}, &httpx.APIError{Status: http.StatusBadRequest, Code: "carga_no_factible",
			Message: "la carga no se puede llevar con este vehículo",
			Details: map[string]any{"motivos": nf.Motivos}}
	case errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil:
		return calculo{}, errCalculoDemoradoAPI
	case err != nil:
		return calculo{}, fmt.Errorf("planificar viajes: %w", err)
	}
	// Datos para evaluar la estimación (ALGORITMO_VIAJES_EMPAQUETADO.md §8).
	slog.InfoContext(ctx, "viajes planificados", "solicitud", solicitudID, "vehiculo", req.VehiculoID,
		"viajes", len(plan.Viajes), "cota_inferior", plan.CotaInferior)
	if len(plan.RotacionNoAplicada) > 0 {
		slog.InfoContext(ctx, "restricción de rotación no aplicada", "solicitud", solicitudID,
			"objetos", plan.RotacionNoAplicada)
	}

	r, err := s.ruta.Calcular(ctx, e.Origen, e.Destino)
	if errors.Is(err, ruteo.ErrRutaNoDisponible) {
		return calculo{}, errRutaAPI
	}
	if err != nil {
		return calculo{}, fmt.Errorf("calcular ruta: %w", err)
	}

	d := cotizar(e.Parametros, e.Costo, plan.Viajes, Ruta(r), e.AccesoOrigen, e.AccesoDestino,
		req.CantidadAyudantes, 0)
	return calculo{Viajes: len(plan.Viajes), Desglose: d}, nil
}

// Cotizar devuelve el precio y los viajes que tendría la oferta, sin guardarla,
// para que el Transportista decida antes de ofertar. Devuelve los mismos errores
// que Crear salvo oferta_duplicada.
func (s *Service) Cotizar(ctx context.Context, id database.Identity, solicitudID string, req OfertaRequest) (CotizacionResponse, error) {
	c, err := s.calcular(ctx, id, solicitudID, req)
	if err != nil {
		return CotizacionResponse{}, err
	}
	return CotizacionResponse{
		CantidadViajes: c.Viajes, CantidadAyudantes: req.CantidadAyudantes,
		PrecioCalculado: dec(c.Desglose.PrecioFinal), Desglose: desgloseResponse(c.Desglose),
	}, nil
}

// Crear calcula y guarda la oferta del Transportista para la solicitud (RF-17).
// Devuelve no_es_transportista, transportista_no_habilitado,
// transportista_no_disponible, solicitud_no_disponible, vehiculo_no_encontrado,
// vehiculo_inactivo, carga_no_factible (con los motivos),
// calculo_demorado, ruta_no_disponible u oferta_duplicada. Escribe oferta y
// oferta_costo.
func (s *Service) Crear(ctx context.Context, id database.Identity, solicitudID string, req OfertaRequest) (OfertaResponse, error) {
	c, err := s.calcular(ctx, id, solicitudID, req)
	if err != nil {
		return OfertaResponse{}, err
	}
	nuevoID, err := s.repo.crear(ctx, id, alta{
		SolicitudID: solicitudID, VehiculoID: req.VehiculoID, Ayudantes: req.CantidadAyudantes,
		Viajes: c.Viajes, Desglose: c.Desglose,
	})
	if errors.Is(err, errDuplicada) {
		return OfertaResponse{}, errDuplicadaAPI
	}
	if err != nil {
		return OfertaResponse{}, err
	}
	return s.detalle(ctx, id, nuevoID)
}

// Retirar retira una oferta pendiente del Transportista (D-23). Devuelve
// oferta_no_encontrada u oferta_no_retirable. Escribe en la base.
func (s *Service) Retirar(ctx context.Context, id database.Identity, ofertaID string) (OfertaResponse, error) {
	err := s.repo.retirar(ctx, id, ofertaID)
	switch {
	case errors.Is(err, errNoEncontrada):
		return OfertaResponse{}, errNoEncontradaAPI
	case errors.Is(err, errNoRetirable):
		return OfertaResponse{}, errNoRetirableAPI
	case err != nil:
		return OfertaResponse{}, err
	}
	return s.detalle(ctx, id, ofertaID)
}

// Propias lista las ofertas del Transportista con su desglose. Devuelve
// no_es_transportista.
func (s *Service) Propias(ctx context.Context, id database.Identity) ([]OfertaResponse, error) {
	p, err := s.repo.perfil(ctx, id)
	if err != nil {
		return nil, err
	}
	if !p.Existe {
		return nil, errNoEsTransportista
	}
	return s.repo.propias(ctx, id, s.hoy())
}

func (s *Service) detalle(ctx context.Context, id database.Identity, ofertaID string) (OfertaResponse, error) {
	o, err := s.repo.detalle(ctx, id, ofertaID, s.hoy())
	if errors.Is(err, errNoEncontrada) {
		return OfertaResponse{}, errNoEncontradaAPI
	}
	return o, err
}

// dec pasa un valor del cálculo a decimal con 2 posiciones, como las columnas de
// oferta y oferta_costo.
func dec(x float64) decimal.Decimal {
	return decimal.MustParse(strconv.FormatFloat(redondear2(x), 'f', 2, 64))
}

func desgloseResponse(d Desglose) DesgloseResponse {
	return DesgloseResponse{
		DistanciaKm: dec(d.DistanciaKm), DuracionRutaH: dec(d.DuracionRutaH),
		DuracionOperacionH: dec(d.DuracionOperacionH), CostoLaboral: dec(d.CostoLaboral),
		CostoVehiculo: dec(d.CostoVehiculo), CostosAdicionales: dec(d.CostosAdicionales),
		CostoOperativo: dec(d.CostoOperativo), MargenPct: dec(d.MargenPct), PrecioNeto: dec(d.PrecioNeto),
		PorcentajeComision: dec(d.ComisionPct), IvaPct: dec(d.IvaPct),
	}
}
