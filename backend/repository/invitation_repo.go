package repository

import (
	"context"
	"log"
	"time"

	"servidor-angular/config"
	"servidor-angular/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InvitationRepository handles the invitations collection.
type InvitationRepository struct {
	collection *mongo.Collection
}

// NewInvitationRepository creates the repository and its unique index so a
// same student can never get two invitations from the same group.
func NewInvitationRepository() *InvitationRepository {
	collection := config.GetCollection("invitations")
	repo := &InvitationRepository{collection: collection}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	index := mongo.IndexModel{
		Keys: bson.D{
			{Key: "groupId", Value: 1},
			{Key: "studentId", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}
	if _, err := collection.Indexes().CreateOne(ctx, index); err != nil {
		log.Printf("No se pudo crear el índice de invitaciones: %v", err)
	}
	return repo
}

// CreateMany inserts invitations in batch.
func (r *InvitationRepository) CreateMany(invitations []models.Invitation) error {
	if len(invitations) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	docs := make([]interface{}, 0, len(invitations))
	for i := range invitations {
		docs = append(docs, invitations[i])
	}
	_, err := r.collection.InsertMany(ctx, docs)
	return err
}

// FindByStudent devuelve todas las invitaciones recibidas por un alumno.
func (r *InvitationRepository) FindByStudent(studentID primitive.ObjectID) ([]models.Invitation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return r.findAll(ctx, bson.M{"studentId": studentID}, bson.D{{Key: "createdAt", Value: -1}})
}

// FindByGroup devuelve las invitaciones enviadas por un grupo.
func (r *InvitationRepository) FindByGroup(groupID primitive.ObjectID) ([]models.Invitation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return r.findAll(ctx, bson.M{"groupId": groupID}, bson.D{{Key: "createdAt", Value: -1}})
}

// FindPendingByID busca una invitación por ID que además esté pendiente.
func (r *InvitationRepository) FindPendingByID(id primitive.ObjectID) (*models.Invitation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var invitation models.Invitation
	err := r.collection.FindOne(ctx, bson.M{"_id": id, "status": models.InvitationPending}).Decode(&invitation)
	if err != nil {
		return nil, err
	}
	return &invitation, nil
}

// SetStatus actualiza el estado de una invitación.
func (r *InvitationRepository) SetStatus(id primitive.ObjectID, status string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"status": status}})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// CancelPendingForStudent cancela todas las invitaciones pendientes de un
// alumno salvo la del grupo que acaba de aceptar. Devuelve cuántas canceló.
func (r *InvitationRepository) CancelPendingForStudent(studentID, exceptGroupID primitive.ObjectID) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"studentId": studentID,
		"status":    models.InvitationPending,
		"groupId":   bson.M{"$ne": exceptGroupID},
	}
	result, err := r.collection.UpdateMany(ctx, filter, bson.M{
		"$set": bson.M{"status": models.InvitationCanceled},
	})
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

// CancelByGroup marca como canceladas las invitaciones pendientes de un grupo
// (por ejemplo, al borrarlo desde el panel).
func (r *InvitationRepository) CancelByGroup(groupID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.UpdateMany(ctx,
		bson.M{"groupId": groupID, "status": models.InvitationPending},
		bson.M{"$set": bson.M{"status": models.InvitationCanceled}},
	)
	return err
}

// HasPendingFromStudent verifica si el alumno ya tiene invitación pendiente
// de un grupo concreto.
func (r *InvitationRepository) HasPendingFromStudent(studentID, groupID primitive.ObjectID) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := r.collection.CountDocuments(ctx, bson.M{
		"studentId": studentID,
		"groupId":   groupID,
		"status":    models.InvitationPending,
	})
	return count > 0, err
}

func (r *InvitationRepository) findAll(ctx context.Context, filter bson.M, sort bson.D) ([]models.Invitation, error) {
	opts := options.Find().SetSort(sort)
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var invitations []models.Invitation
	if err := cursor.All(ctx, &invitations); err != nil {
		return nil, err
	}
	if invitations == nil {
		invitations = []models.Invitation{}
	}
	return invitations, nil
}
