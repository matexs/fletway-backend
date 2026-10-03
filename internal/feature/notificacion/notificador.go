// Package notificacion crea las notificaciones in-app de los usuarios (RF-09).
//
// D-22: en esta etapa toda notificación es una fila en notificacion. El envío pasa
// por la interfaz Notificador para que sumar push (FCM) más adelante sea otra
// implementación, sin tocar a quienes notifican.
package notificacion

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/matexs/fletway-backend/internal/platform/database"
)

// Tipos de notificación (tipo_notificacion.codigo). Los textos están en
// docs/PLAN_CONSTRUCCION.md §3.
const (
	// TipoDocumentacionRevisada avisa al Transportista el resultado de la revisión
	// de su documentación (RF-01).
	TipoDocumentacionRevisada = "documentacion_revisada"
)

// ErrSinDestinatario se devuelve si la notificación no tiene destinatario o texto.
var ErrSinDestinatario = errors.New("notificacion: falta destinatario o contenido")

// Nueva es una notificación a crear.
type Nueva struct {
	// DestinatarioID es usuario.id de quien la recibe.
	DestinatarioID string
	// Tipo es un tipo_notificacion.codigo (ver las constantes Tipo*).
	Tipo string
	// Contenido es el texto que ve el usuario.
	Contenido string
	// EntidadTipo y EntidadID referencian el objeto que la originó (opcionales),
	// para que la app pueda abrirlo.
	EntidadTipo string
	EntidadID   string
}

// Notificador entrega notificaciones a los usuarios.
type Notificador interface {
	// Notificar entrega n. La identidad id es la de quien la genera: las policies
	// de notificacion deciden si puede crearla.
	Notificar(ctx context.Context, id database.Identity, n Nueva) error
}

// InApp es el Notificador de esta etapa: inserta la fila en notificacion (D-22).
type InApp struct {
	db *database.DB
}

// NewInApp crea el Notificador in-app sobre la base dada.
func NewInApp(db *database.DB) *InApp {
	return &InApp{db: db}
}

// Notificar inserta la notificación con RLS pass-through. Devuelve
// ErrSinDestinatario si faltan datos. Escribe en la base.
func (n *InApp) Notificar(ctx context.Context, id database.Identity, nueva Nueva) error {
	if nueva.DestinatarioID == "" || nueva.Contenido == "" || nueva.Tipo == "" {
		return ErrSinDestinatario
	}
	err := n.db.WithinTx(ctx, id, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO notificacion (destinatario_usuario_id, tipo_notificacion_codigo, contenido,
			                          entidad_referencia_tipo, entidad_referencia_id)
			VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, '')::uuid)`,
			nueva.DestinatarioID, nueva.Tipo, nueva.Contenido, nueva.EntidadTipo, nueva.EntidadID)
		return err
	})
	if err != nil {
		return fmt.Errorf("crear notificación %s: %w", nueva.Tipo, err)
	}
	return nil
}
