// Package httpx tiene helpers de respuesta JSON y el envelope de error único (D-11).
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// ErrorBody es el envelope de error único de la API.
//
//	{ "error": { "code": "...", "message": "...", "details": {} } }
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// APIError es un error con código estable y status HTTP, apto para devolver al cliente.
type APIError struct {
	Status  int
	Code    string
	Message string
	Details map[string]any
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

// Constructores frecuentes.
func BadRequest(code, msg string) *APIError { return &APIError{http.StatusBadRequest, code, msg, nil} }
func Unauthorized(code, msg string) *APIError {
	return &APIError{http.StatusUnauthorized, code, msg, nil}
}
func Forbidden(code, msg string) *APIError { return &APIError{http.StatusForbidden, code, msg, nil} }
func NotFound(code, msg string) *APIError  { return &APIError{http.StatusNotFound, code, msg, nil} }
func Conflict(code, msg string) *APIError  { return &APIError{http.StatusConflict, code, msg, nil} }
func Internal(msg string) *APIError {
	return &APIError{http.StatusInternalServerError, "error_interno", msg, nil}
}

// JSON escribe un cuerpo JSON con el status dado.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("httpx: fallo al serializar respuesta", "err", err)
	}
}

// Error escribe el envelope de error. Si err no es *APIError, responde 500 genérico
// sin filtrar detalles internos.
func Error(w http.ResponseWriter, err error) {
	apiErr, ok := err.(*APIError)
	if !ok {
		apiErr = Internal("ocurrió un error inesperado")
		slog.Error("httpx: error no tipado en handler", "err", err)
	}
	JSON(w, apiErr.Status, ErrorBody{Error: ErrorDetail{
		Code:    apiErr.Code,
		Message: apiErr.Message,
		Details: apiErr.Details,
	}})
}

// Decode parsea el body JSON del request en dst, rechazando campos desconocidos.
func Decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return BadRequest("json_invalido", "el cuerpo del request no es JSON válido: "+err.Error())
	}
	return nil
}

// Handler es un http.HandlerFunc que puede devolver error; Wrap lo adapta.
type Handler func(w http.ResponseWriter, r *http.Request) error

// Wrap convierte un Handler en http.HandlerFunc, canalizando el error al envelope.
func Wrap(h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			Error(w, err)
		}
	}
}
