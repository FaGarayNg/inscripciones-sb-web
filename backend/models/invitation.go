package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Estados posibles de una invitación.
const (
	InvitationPending  = "PENDING"
	InvitationAccepted = "ACCEPTED"
	InvitationRejected = "REJECTED"
	InvitationCanceled = "CANCELED"
)

// Invitation es una invitación de un grupo a un ingresante. Vive en su propia
// colección: antes estaba embebida en el alumno, lo que hacía difícil listar
// las invitaciones de cada persona y cancelarlas en bloque.
type Invitation struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	GroupID     primitive.ObjectID `json:"groupId" bson:"groupId"`
	StudentID   primitive.ObjectID `json:"studentId" bson:"studentId"`
	StudentName string             `json:"studentName" bson:"studentName"`
	LeaderName  string             `json:"leaderName" bson:"leaderName"`
	Status      string             `json:"status" bson:"status"`
	CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
}
