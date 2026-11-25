package repository

import (
	"context"
	"time"

	"servidor-angular/config"
	"servidor-angular/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GroupRepository handles database operations for groups. El grupo guarda
// solo IDs: las fichas de los alumnos viven en la colección students.
type GroupRepository struct {
	collection *mongo.Collection
}

// NewGroupRepository creates a new group repository
func NewGroupRepository() *GroupRepository {
	return &GroupRepository{
		collection: config.GetCollection("groups"),
	}
}

// FindAll returns all groups
func (r *GroupRepository) FindAll() ([]models.Group, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var groups []models.Group
	if err = cursor.All(ctx, &groups); err != nil {
		return nil, err
	}
	if groups == nil {
		groups = []models.Group{}
	}
	return groups, nil
}

// FindByID returns a group by ID
func (r *GroupRepository) FindByID(id primitive.ObjectID) (*models.Group, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var group models.Group
	if err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&group); err != nil {
		return nil, err
	}
	return &group, nil
}

// FindByLeader devuelve el grupo que lidera un alumno, si tiene uno.
func (r *GroupRepository) FindByLeader(leaderID primitive.ObjectID) (*models.Group, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var group models.Group
	if err := r.collection.FindOne(ctx, bson.M{"leaderId": leaderID}).Decode(&group); err != nil {
		return nil, err
	}
	return &group, nil
}

// Create inserts a new group
func (r *GroupRepository) Create(group *models.Group) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.collection.InsertOne(ctx, group)
	if err != nil {
		return err
	}
	group.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// AddMember incorpora un alumno al grupo respetando el cupo de forma atómica.
// El filtro con $expr impide que dos invitados acepten el mismo lugar último.
// Devuelve mongo.ErrNoDocuments si el grupo no existe, es solitario, está lleno
// o el alumno ya está dentro.
func (r *GroupRepository) AddMember(groupID, studentID primitive.ObjectID) (*models.Group, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"_id":       groupID,
		"solo":      false,
		"memberIds": bson.M{"$ne": studentID},
		"leaderId":  bson.M{"$ne": studentID},
		"$expr": bson.M{
			"$lt": []interface{}{bson.M{"$size": "$memberIds"}, models.MaxGroupSize - 1},
		},
	}
	pipeline := []bson.M{
		{"$set": bson.M{
			"memberIds": bson.M{"$concatArrays": []interface{}{
				"$memberIds", []primitive.ObjectID{studentID},
			}},
		}},
		{"$set": bson.M{
			"full": bson.M{"$eq": []interface{}{bson.M{"$size": "$memberIds"}, models.MaxGroupSize}},
		}},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var group models.Group
	if err := r.collection.FindOneAndUpdate(ctx, filter, pipeline, opts).Decode(&group); err != nil {
		return nil, err
	}
	return &group, nil
}

// RemoveMember quita un alumno del grupo y libera el lugar.
func (r *GroupRepository) RemoveMember(groupID, studentID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": groupID, "memberIds": studentID}
	update := bson.M{
		"$pull": bson.M{"memberIds": studentID},
		"$set":  bson.M{"full": false},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// Delete removes a group
func (r *GroupRepository) Delete(id primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
