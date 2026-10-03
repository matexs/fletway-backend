// Package dbtest tiene ayudas para los tests de integración contra Supabase local
// (D-16, D-12). Nunca se usa contra dbFletway: la base sale de TEST_DATABASE_URL, que
// apunta al contenedor local, y si no está definida el test se saltea.
package dbtest

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/platform/config"
	"github.com/matexs/fletway-backend/internal/platform/database"
)

// URL devuelve TEST_DATABASE_URL o saltea el test si no está definida.
func URL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL no definida: test de integración salteado (requiere Supabase local)")
	}
	return url
}

// Abrir abre un *database.DB contra la base local y lo cierra al terminar el test.
func Abrir(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.Open(context.Background(), config.Config{
		Database: config.DatabaseConfig{URL: URL(t), MaxConns: 4, MinConns: 0},
	})
	require.NoError(t, err)
	t.Cleanup(db.Close)
	return db
}

// Exec corre una sentencia como superusuario, sin RLS, y falla el test si da error.
// Sirve para preparar datos que en producción crea otra vía (Supabase Auth, un
// Administrador).
func Exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	require.NoError(t, Intentar(t, sql, args...))
}

// Intentar corre una sentencia como superusuario, sin RLS, y devuelve su error. Sirve
// para comprobar que la base rechaza algo (un trigger, un constraint).
func Intentar(t *testing.T, sql string, args ...any) error {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, URL(t))
	require.NoError(t, err)
	defer func() { _ = conn.Close(ctx) }()
	_, err = conn.Exec(ctx, sql, args...)
	return err
}

// Identidad arma la identidad que el middleware Auth pondría en el contexto para sub.
func Identidad(t *testing.T, sub string) database.Identity {
	t.Helper()
	raw, err := json.Marshal(map[string]string{"sub": sub, "role": "authenticated", "aud": "authenticated"})
	require.NoError(t, err)
	return database.Identity{UserID: sub, Role: "authenticated", RawClaims: string(raw)}
}

// CrearUsuario simula el signUp de Supabase Auth (D-18): inserta en auth.users con la
// metadata dada, lo que dispara trg_alta_usuario. Devuelve la identidad del usuario
// nuevo y lo borra al terminar el test (en cascada borra usuario y su fila de rol).
func CrearUsuario(t *testing.T, rol string) database.Identity {
	t.Helper()
	id := nuevoUUID(t)
	meta, err := json.Marshal(map[string]string{
		"rol":             rol,
		"nombre_completo": "Usuario de Prueba",
		"telefono":        "1155550000",
	})
	require.NoError(t, err)
	Exec(t, `INSERT INTO auth.users (id, email, aud, role, raw_user_meta_data)
		VALUES ($1, $2, 'authenticated', 'authenticated', $3)`, id, id+"@prueba.fletway.local", meta)
	t.Cleanup(func() { Exec(t, `DELETE FROM auth.users WHERE id = $1`, id) })
	return Identidad(t, id)
}

// nuevoUUID genera un UUID v4 sin sumar una dependencia sólo para los tests.
func nuevoUUID(t *testing.T) string {
	t.Helper()
	var b [16]byte
	_, err := rand.Read(b[:])
	require.NoError(t, err)
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
