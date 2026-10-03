package identidad

import "github.com/matexs/fletway-backend/internal/platform/httpx"

// MeResponse es el perfil del usuario autenticado que devuelven GET /api/me y los
// endpoints de registro. Es la fuente del rol para la app (D-18): nunca se usa el rol
// de user_metadata.
type MeResponse struct {
	UsuarioID      string `json:"usuario_id"`
	Email          string `json:"email"`
	NombreCompleto string `json:"nombre_completo"`
	Telefono       string `json:"telefono"`
	// Rol es cliente, transportista o administrador.
	Rol    string `json:"rol"`
	Activo bool   `json:"activo"`
	// RegistroCompleto es false mientras falte la fila del rol (cliente, transportista o
	// administrador): la app tiene que llamar al endpoint de registro que corresponda.
	RegistroCompleto bool `json:"registro_completo"`
	// EstadoHabilitacion es el estado del Transportista (pendiente, habilitado, ...); nil
	// para los otros roles o si el registro no está completo.
	EstadoHabilitacion *string `json:"estado_habilitacion"`
	// Disponible es el interruptor "estoy tomando trabajos" del Transportista
	// (D-21); nil para los otros roles o si el registro no está completo.
	Disponible *bool `json:"disponible"`
}

// DisponibilidadRequest prende o apaga la disponibilidad del Transportista.
type DisponibilidadRequest struct {
	Disponible *bool `json:"disponible"`
}

// Validate exige el campo disponible.
func (r *DisponibilidadRequest) Validate() error {
	if r.Disponible == nil {
		return httpx.BadRequest("datos_incompletos", "falta disponible")
	}
	return nil
}

// toMeResponse convierte el perfil en la respuesta; los campos coinciden uno a uno.
func toMeResponse(p Perfil) MeResponse {
	return MeResponse(p)
}
