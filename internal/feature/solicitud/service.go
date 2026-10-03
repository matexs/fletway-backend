package solicitud

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/geocodificacion"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// horaArgentina es la zona horaria del negocio (UTC-3, sin horario de verano):
// "hoy" y "vencida" se cuentan en hora argentina, no en la del servidor.
var horaArgentina = time.FixedZone("ART", -3*60*60)

var (
	errNoEsCliente = httpx.Forbidden("no_es_cliente",
		"sólo un Cliente registrado puede publicar y ver sus solicitudes")
	errSolicitudNoEncontrada = httpx.NotFound("solicitud_no_encontrada", "no existe la solicitud o no es tuya")
	errFechaPasada           = httpx.BadRequest("fecha_pasada", "la fecha del servicio no puede ser anterior a hoy")
	errZonaInvalida          = httpx.BadRequest("zona_invalida", "alguna de las zonas no existe")
	errObjetoInvalido        = httpx.BadRequest("objeto_invalido", "algún objeto del catálogo no existe")
	errDireccion             = httpx.BadRequest("direccion_no_ubicable",
		"no pudimos ubicar la dirección en la zona elegida")
	errNoCancelableAPI = httpx.Conflict("solicitud_no_cancelable", "sólo se cancela una solicitud publicada")
	errNoVencidaAPI    = httpx.Conflict("solicitud_no_vencida", "sólo se republica una solicitud vencida")
)

// Service publica, lista, cancela y republica solicitudes (RF-06, D-20).
type Service struct {
	repo  *repository
	geo   geocodificacion.Geocodificador
	ahora func() time.Time
}

// NewService crea el Service. geo ubica las direcciones (D-20).
func NewService(db *database.DB, geo geocodificacion.Geocodificador) *Service {
	return &Service{repo: &repository{db: db}, geo: geo, ahora: time.Now}
}

// ConReloj reemplaza el reloj del Service; sirve para los tests del vencimiento.
func (s *Service) ConReloj(ahora func() time.Time) *Service {
	s.ahora = ahora
	return s
}

func (s *Service) hoy() time.Time {
	a := s.ahora().In(horaArgentina)
	return time.Date(a.Year(), a.Month(), a.Day(), 0, 0, 0, 0, time.UTC)
}

// Crear publica una solicitud (RF-06): ubica origen y destino, copia los objetos
// del catálogo (lo hace la base, RN-08) y no calcula ningún monto (RN-01).
// Devuelve no_es_cliente, fecha_pasada, zona_invalida, objeto_invalido o
// direccion_no_ubicable. Escribe en la base.
func (s *Service) Crear(ctx context.Context, id database.Identity, req CrearSolicitudRequest) (SolicitudResponse, error) {
	if err := s.exigirCliente(ctx, id); err != nil {
		return SolicitudResponse{}, err
	}
	if req.Fecha().Before(s.hoy()) {
		return SolicitudResponse{}, errFechaPasada
	}
	zonas, err := s.repo.zonas(ctx, id, req.Origen.ZonaID, req.Destino.ZonaID)
	if err != nil {
		return SolicitudResponse{}, err
	}
	zo, okO := zonas[req.Origen.ZonaID]
	zd, okD := zonas[req.Destino.ZonaID]
	if !okO || !okD {
		return SolicitudResponse{}, errZonaInvalida
	}
	origen, err := s.geo.Geocodificar(ctx, req.Origen.Direccion, zo.Nombre, zo.Provincia)
	if err != nil {
		return SolicitudResponse{}, s.traducirGeo(err)
	}
	destino, err := s.geo.Geocodificar(ctx, req.Destino.Direccion, zd.Nombre, zd.Provincia)
	if err != nil {
		return SolicitudResponse{}, s.traducirGeo(err)
	}
	nuevoID, err := s.repo.crear(ctx, id, alta{
		Origen: req.Origen, Destino: req.Destino, OrigenCoord: origen, DestinoCoord: destino,
		Fecha: req.Fecha(), FranjaInicio: req.FranjaHorariaInicio, FranjaFin: req.FranjaHorariaFin,
		Ayudantes: req.CantidadAyudantesSolicitados, Objetos: req.Objetos,
	})
	if errors.Is(err, errObjetoInexistente) {
		return SolicitudResponse{}, errObjetoInvalido
	}
	if err != nil {
		return SolicitudResponse{}, err
	}
	return s.Detalle(ctx, id, nuevoID)
}

func (s *Service) traducirGeo(err error) error {
	if errors.Is(err, geocodificacion.ErrNoGeocodificable) {
		return errDireccion
	}
	return fmt.Errorf("ubicar dirección: %w", err)
}

// Detalle devuelve una solicitud visible para la identidad. Devuelve
// solicitud_no_encontrada.
func (s *Service) Detalle(ctx context.Context, id database.Identity, solicitudID string) (SolicitudResponse, error) {
	out, err := s.repo.detalle(ctx, id, solicitudID, s.hoy())
	if errors.Is(err, errNoEncontrada) {
		return SolicitudResponse{}, errSolicitudNoEncontrada
	}
	return out, err
}

// Propias lista las solicitudes del Cliente con el estado al día (vencida se
// calcula al leer, D-20). Devuelve no_es_cliente.
func (s *Service) Propias(ctx context.Context, id database.Identity) ([]SolicitudResumen, error) {
	if err := s.exigirCliente(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.propias(ctx, id, s.hoy())
}

// Cancelar cancela una solicitud publicada del Cliente, sin costo: todavía no
// hay compromiso (D-20). Sus ofertas pendientes pasan a no_seleccionada.
// Devuelve solicitud_no_encontrada o solicitud_no_cancelable. Escribe en la base.
func (s *Service) Cancelar(ctx context.Context, id database.Identity, solicitudID string) (SolicitudResponse, error) {
	err := s.repo.cancelar(ctx, id, solicitudID)
	switch {
	case errors.Is(err, errNoEncontrada):
		return SolicitudResponse{}, errSolicitudNoEncontrada
	case errors.Is(err, errNoCancelable):
		return SolicitudResponse{}, errNoCancelableAPI
	case err != nil:
		return SolicitudResponse{}, err
	}
	return s.Detalle(ctx, id, solicitudID)
}

// Republicar crea una solicitud nueva a partir de una vencida del Cliente, con
// otra fecha (D-20). Devuelve fecha_pasada, solicitud_no_encontrada o
// solicitud_no_vencida. Escribe en la base.
func (s *Service) Republicar(ctx context.Context, id database.Identity, solicitudID string, req RepublicarRequest) (SolicitudResponse, error) {
	hoy := s.hoy()
	if req.fecha.Before(hoy) {
		return SolicitudResponse{}, errFechaPasada
	}
	nuevoID, err := s.repo.republicar(ctx, id, solicitudID, hoy, req.fecha,
		req.FranjaHorariaInicio, req.FranjaHorariaFin)
	switch {
	case errors.Is(err, errNoEncontrada):
		return SolicitudResponse{}, errSolicitudNoEncontrada
	case errors.Is(err, errNoVencida):
		return SolicitudResponse{}, errNoVencidaAPI
	case err != nil:
		return SolicitudResponse{}, err
	}
	return s.Detalle(ctx, id, nuevoID)
}

func (s *Service) exigirCliente(ctx context.Context, id database.Identity) error {
	es, err := s.repo.esCliente(ctx, id)
	if err != nil {
		return err
	}
	if !es {
		return errNoEsCliente
	}
	return nil
}
