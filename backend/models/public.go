package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PublicStudent es la vista de un alumno que se expone sin autenticación.
// No incluye el correo electrónico (dato personal que el ingresante carga al
// confirmar) ni la información de grupos/invitaciones.
type PublicStudent struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	DNI       string             `json:"dni" bson:"dni"`
	Name      string             `json:"name" bson:"name"`
	Instance  string             `json:"instance" bson:"instance"`
	Title     string             `json:"title" bson:"title"`
	School    string             `json:"school" bson:"school"`
	Confirmed bool               `json:"confirmed" bson:"confirmed"`
	GroupID   string             `json:"groupID,omitempty" bson:"groupID,omitempty"`
}

// ToPublicStudent convierte un Student a su vista pública.
func ToPublicStudent(student Student) PublicStudent {
	return PublicStudent{
		ID:        student.ID,
		DNI:       student.DNI,
		Name:      student.Name,
		Instance:  student.Instance,
		Title:     student.Title,
		School:    student.School,
		Confirmed: student.Confirmed,
		GroupID:   student.GroupID,
	}
}

// ToPublicStudents convierte una lista de alumnos a sus vistas públicas.
func ToPublicStudents(students []Student) []PublicStudent {
	public := make([]PublicStudent, 0, len(students))
	for _, student := range students {
		public = append(public, ToPublicStudent(student))
	}
	return public
}

// PublicGroup es la vista de un grupo: como el grupo guarda solo IDs, el
// handler arma leader/members consultando los alumnos. Nunca incluye emails.
type PublicGroup struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Leader    PublicStudent      `json:"leader" bson:"leader"`
	Members   []PublicStudent    `json:"members" bson:"members"`
	Solo      bool               `json:"solo" bson:"solo"`
	Full      bool               `json:"full" bson:"full"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
}

// NewPublicGroup arma la vista pública a partir del grupo y de los alumnos
// referenciados. Los ids que no se encuentran se omiten.
func NewPublicGroup(group Group, students map[primitive.ObjectID]Student) PublicGroup {
	leader := students[group.LeaderID]
	members := make([]PublicStudent, 0, len(group.MemberIDs))
	for _, id := range group.MemberIDs {
		if student, ok := students[id]; ok {
			members = append(members, ToPublicStudent(student))
		}
	}

	return PublicGroup{
		ID:        group.ID,
		Leader:    ToPublicStudent(leader),
		Members:   members,
		Solo:      group.Solo,
		Full:      group.Full,
		CreatedAt: group.CreatedAt,
	}
}
