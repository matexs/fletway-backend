package catalogo_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/matexs/fletway-backend/internal/feature/catalogo"
	"github.com/matexs/fletway-backend/internal/platform/auth"
	"github.com/matexs/fletway-backend/internal/platform/database/dbtest"
)

func TestObjetos(t *testing.T) {
	db := dbtest.Abrir(t)
	mux := http.NewServeMux()
	catalogo.Register(mux, db)

	for _, rol := range []string{"cliente", "transportista"} {
		t.Run("lo lee un "+rol, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/catalogo/objetos", nil)
			r = r.WithContext(auth.WithIdentity(r.Context(), dbtest.CrearUsuario(t, rol)))
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			require.Equal(t, http.StatusOK, w.Code)

			var objetos []catalogo.ObjetoResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &objetos))
			require.Len(t, objetos, 28, "el catálogo de la migración 0009")
			assert.True(t, sort.SliceIsSorted(objetos, func(i, j int) bool { return objetos[i].Nombre < objetos[j].Nombre }))
			assert.NotContains(t, w.Body.String(), "volumen_estimado_m3", "columna deprecada")

			var heladera *catalogo.ObjetoResponse
			for i := range objetos {
				assert.Positive(t, objetos[i].LargoM.Sign(), objetos[i].Nombre)
				if objetos[i].Nombre == "Heladera" {
					heladera = &objetos[i]
				}
			}
			require.NotNil(t, heladera)
			assert.Equal(t, "1.8", heladera.AltoM.String())
			assert.False(t, heladera.RotacionVertical, "una heladera no se acuesta")
			assert.False(t, heladera.Apilable)
		})
	}
}
