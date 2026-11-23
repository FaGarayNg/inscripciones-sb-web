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

// AbanderadoRepository handles database operations for the abanderados list.
type AbanderadoRepository struct {
	collection *mongo.Collection
}

// NewAbanderadoRepository creates a new abanderado repository.
func NewAbanderadoRepository() *AbanderadoRepository {
	return &AbanderadoRepository{
		collection: config.GetCollection("abanderados"),
	}
}

// FindAll devuelve la lista completa, ordenada por nombre para que el panel
// admin muestre un listado estable.
func (r *AbanderadoRepository) FindAll() ([]models.Abanderado, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var abanderados []models.Abanderado
	if err = cursor.All(ctx, &abanderados); err != nil {
		return nil, err
	}

	if abanderados == nil {
		abanderados = []models.Abanderado{}
	}

	return abanderados, nil
}

// FindAllDNIs devuelve los DNI ya cargados, para no duplicar abanderados al
// reimportar una planilla.
func (r *AbanderadoRepository) FindAllDNIs() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	values, err := r.collection.Distinct(ctx, "dni", bson.M{})
	if err != nil {
		return nil, err
	}

	dnIs := make([]string, 0, len(values))
	for _, value := range values {
		if dni, ok := value.(string); ok {
			dnIs = append(dnIs, dni)
		}
	}

	return dnIs, nil
}

// Create inserta un abanderado y le asigna el ID generado por MongoDB.
func (r *AbanderadoRepository) Create(abanderado *models.Abanderado) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := r.collection.InsertOne(ctx, abanderado)
	if err != nil {
		return err
	}

	abanderado.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// Delete elimina un abanderado por ID.
func (r *AbanderadoRepository) Delete(id primitive.ObjectID) error {
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
