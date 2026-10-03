// Package storage habla con la API de Supabase Storage en nombre del usuario (D-19).
//
// Los archivos se suben desde la app directo a Storage; el backend sólo pide URLs
// firmadas para leerlos. Cada llamada usa el JWT del usuario, así que las policies
// de storage.objects siguen decidiendo quién ve qué (no se usa la service_role key).
package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/matexs/fletway-backend/internal/platform/config"
)

// ErrSinConfig indica que falta SUPABASE_URL o SUPABASE_ANON_KEY.
var ErrSinConfig = errors.New("storage: faltan SUPABASE_URL o SUPABASE_ANON_KEY")

// Firmador genera URLs firmadas de lectura. Es la interfaz que consumen los
// services, para poder reemplazarla en los tests.
type Firmador interface {
	// FirmarURL devuelve una URL de lectura de bucket/path que vence en vence,
	// pedida con el JWT token del usuario. Falla si las policies de Storage no le
	// dejan leer el objeto o si el objeto no existe.
	FirmarURL(ctx context.Context, token, bucket, path string, vence time.Duration) (string, error)
}

// Cliente es el Firmador que llama a la API REST de Supabase Storage.
type Cliente struct {
	base   string
	apikey string
	http   *http.Client
}

// New crea el cliente con la URL y la clave pública del proyecto. Devuelve
// ErrSinConfig si falta alguna.
func New(cfg config.SupabaseConfig) (*Cliente, error) {
	if cfg.URL == "" || cfg.AnonKey == "" {
		return nil, ErrSinConfig
	}
	return &Cliente{
		base:   strings.TrimRight(cfg.URL, "/") + "/storage/v1",
		apikey: cfg.AnonKey,
		http:   &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// FirmarURL implementa Firmador con POST /object/sign/{bucket}/{path}.
func (c *Cliente) FirmarURL(ctx context.Context, token, bucket, path string, vence time.Duration) (string, error) {
	cuerpo, err := json.Marshal(map[string]int{"expiresIn": int(vence.Seconds())})
	if err != nil {
		return "", fmt.Errorf("firmar url: %w", err)
	}
	endpoint := c.base + "/object/sign/" + url.PathEscape(bucket) + "/" + escaparPath(path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(cuerpo))
	if err != nil {
		return "", fmt.Errorf("firmar url: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", c.apikey)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("firmar url: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		detalle, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return "", fmt.Errorf("firmar url: storage respondió %d: %s", res.StatusCode, detalle)
	}
	var out struct {
		SignedURL string `json:"signedURL"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("firmar url: leer respuesta: %w", err)
	}
	if out.SignedURL == "" {
		return "", errors.New("firmar url: respuesta sin signedURL")
	}
	return c.base + out.SignedURL, nil
}

// escaparPath escapa cada segmento del path del objeto, sin tocar las barras.
func escaparPath(p string) string {
	partes := strings.Split(p, "/")
	for i, s := range partes {
		partes[i] = url.PathEscape(s)
	}
	return strings.Join(partes, "/")
}
