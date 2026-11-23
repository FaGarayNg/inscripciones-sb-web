package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Abanderado representa a un abanderado o escolta titular cargado desde la
// planilla de la secretaría. El curso se guarda como texto libre (por ejemplo,
// "1° A") porque puede cambiar entre ciclos lectivos.
type Abanderado struct {
	ID    primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name  string             `json:"name" bson:"name"`
	DNI   string             `json:"dni" bson:"dni"`
	Curso string             `json:"curso" bson:"curso"`
}
