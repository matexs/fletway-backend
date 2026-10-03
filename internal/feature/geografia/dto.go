package geografia

import (
	"regexp"

	"github.com/matexs/fletway-backend/internal/platform/httpx"
)

// maxZonas acota la selección: el catálogo piloto tiene 14 zonas (D-20).
const maxZonas = 50

var uuidValido = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ZonaResponse es una zona del catálogo.
type ZonaResponse struct {
	ID        string `json:"id"`
	Nombre    string `json:"nombre"`
	Provincia string `json:"provincia"`
}

// ZonasTransportista son las zonas de trabajo elegidas (request y response de
// /transportista/zonas).
type ZonasTransportista struct {
	ZonaIDs []string `json:"zona_ids"`
}

// Validate exige una lista (puede ser vacía) de ids válidos, sin repetidos y de
// hasta 50. Devuelve un *httpx.APIError (datos_incompletos, datos_invalidos o
// zona_invalida).
func (z *ZonasTransportista) Validate() error {
	if z.ZonaIDs == nil {
		return httpx.BadRequest("datos_incompletos", "falta zona_ids")
	}
	if len(z.ZonaIDs) > maxZonas {
		return httpx.BadRequest("datos_invalidos", "no se pueden elegir más de 50 zonas")
	}
	vistos := map[string]bool{}
	for _, id := range z.ZonaIDs {
		if !uuidValido.MatchString(id) {
			return errZonaInvalida
		}
		if vistos[id] {
			return httpx.BadRequest("datos_invalidos", "zona_ids tiene zonas repetidas")
		}
		vistos[id] = true
	}
	return nil
}
