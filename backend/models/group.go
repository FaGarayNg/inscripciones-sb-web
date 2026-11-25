package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MaxGroupSize es el cupo máximo por grupo, contando al líder.
const MaxGroupSize = 3

// Group representa un grupo de ingresantes. Guarda referencias por ID (no
// copias de los alumnos) para que los cambios en una ficha no dejen datos
// desactualizados dentro del grupo.
type Group struct {
	ID        primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	LeaderID  primitive.ObjectID   `json:"leaderId" bson:"leaderId"`
	MemberIDs []primitive.ObjectID `json:"memberIds" bson:"memberIds"`
	Solo      bool                 `json:"solo" bson:"solo"`
	Full      bool                 `json:"full,omitempty" bson:"full,omitempty"`
	CreatedAt time.Time            `json:"createdAt" bson:"createdAt"`
}

// HasMember indica si el alumno ya pertenece al grupo (el líder siempre cuenta).
func (g Group) HasMember(id primitive.ObjectID) bool {
	if g.LeaderID == id {
		return true
	}
	for _, memberID := range g.MemberIDs {
		if memberID == id {
			return true
		}
	}
	return false
}
