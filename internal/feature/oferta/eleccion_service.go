package oferta

import (
	"context"
	"errors"
	"strconv"

	"github.com/matexs/fletway-backend/internal/platform/database"
	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

const (
	// topOfertas es cuántas ofertas ve el Cliente de entrada (RN-05).
	topOfertas = 3
	// limiteVerMas es el tamaño de página por defecto de "ver más".
	limiteVerMas = 10
	// maxLimiteVerMas es el tope del tamaño de página.
	maxLimiteVerMas = 50
)

var (
	errSolicitudClienteAPI  = httpx.NotFound("solicitud_no_encontrada", "no existe la solicitud o no es tuya")
	errOfertaNoPendienteAPI = httpx.Conflict("oferta_no_disponible",
		"la oferta ya no está disponible: el Transportista la retiró o ya elegiste otra")
	errSolicitudNoAsignableAPI = httpx.Conflict("solicitud_no_asignable",
		"la solicitud ya no está publicada o venció")
	errTransportistaBajaAPI = httpx.Conflict("transportista_inhabilitado",
		"el Transportista ya no puede tomar el viaje; elegí otra oferta")
)

// Pagina es la porción pedida del listado ordenado: sin VerMas, las 3
// primeras; con VerMas, Limite ofertas desde Cursor.
type Pagina struct {
	VerMas bool
	Cursor int
	Limite int
}

// OfertasDeSolicitud devuelve las ofertas pendientes de una solicitud del
// Cliente ordenadas por score (RN-05, RF-07), sin desglose ni patente. Devuelve
// solicitud_no_encontrada.
func (s *Service) OfertasDeSolicitud(ctx context.Context, id database.Identity, solicitudID string, p Pagina) (OfertasDeSolicitudResponse, error) {
	pedidos, ofertas, err := s.repo.ofertasDeSolicitud(ctx, id, solicitudID)
	if errors.Is(err, errSolicitudInexistente) {
		return OfertasDeSolicitudResponse{}, errSolicitudClienteAPI
	}
	if err != nil {
		return OfertasDeSolicitudResponse{}, err
	}
	puntuables := make([]Puntuable, len(ofertas))
	for i, o := range ofertas {
		precio, _ := strconv.ParseFloat(o.PrecioCalculado.String(), 64)
		puntuables[i] = Puntuable{OfertaID: o.ID, Precio: precio, Calificacion: o.Calificacion,
			TasaCumplimiento: o.TasaCumplimiento, CreadoEn: o.CreadoEn}
	}
	ordenadas := make([]OfertaParaCliente, 0, len(ofertas))
	for _, i := range ordenarPorScore(puntuables) {
		ordenadas = append(ordenadas, ofertas[i])
	}

	desde, hasta := 0, topOfertas
	if p.VerMas {
		desde, hasta = p.Cursor, p.Cursor+p.Limite
	}
	desde = min(desde, len(ordenadas))
	hasta = min(hasta, len(ordenadas))
	out := OfertasDeSolicitudResponse{
		CantidadAyudantesSolicitados: pedidos,
		Total:                        len(ordenadas),
		Ofertas:                      ordenadas[desde:hasta],
	}
	if hasta < len(ordenadas) {
		out.SiguienteCursor = &hasta
	}
	return out, nil
}

// Aceptar acepta una oferta pendiente de una solicitud del Cliente (RF-07): crea
// el viaje con su desglose y sus PIN, rechaza las demás ofertas y asigna la
// solicitud, todo en fn_aceptar_oferta (D-23). Devuelve oferta_no_encontrada,
// oferta_no_disponible, solicitud_no_asignable o transportista_inhabilitado.
// Escribe en la base.
func (s *Service) Aceptar(ctx context.Context, id database.Identity, ofertaID string) (ViajeConfirmadoResponse, error) {
	v, err := s.repo.aceptar(ctx, id, ofertaID)
	switch {
	case errors.Is(err, errNoEncontrada):
		return ViajeConfirmadoResponse{}, errNoEncontradaAPI
	case errors.Is(err, errOfertaNoPendiente):
		return ViajeConfirmadoResponse{}, errOfertaNoPendienteAPI
	case errors.Is(err, errSolicitudNoAsignable):
		return ViajeConfirmadoResponse{}, errSolicitudNoAsignableAPI
	case errors.Is(err, errTransportistaBaja):
		return ViajeConfirmadoResponse{}, errTransportistaBajaAPI
	}
	return v, err
}
