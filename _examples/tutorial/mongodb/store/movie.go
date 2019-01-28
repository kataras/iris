package store

import (
	"context"
	"errors"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/primitive"
	"github.com/mongodb/mongo-go-driver/mongo"
	// up to you:
	// "github.com/mongodb/mongo-go-driver/mongo/options"
)

type Movie struct {
	ID          primitive.ObjectID `json:"_id" bson:"_id"` /* you need the bson:"_id" to be able to retrieve with ID filled */
	Name        string             `json:"name"`
	Cover       string             `json:"cover"`
	Description string             `json:"description"`
}

type MovieService interface {
	GetAll(ctx context.Context) ([]Movie, error)
	GetByID(ctx context.Context, id string) (Movie, error)
	Create(ctx context.Context, m *Movie) error
	Update(ctx context.Context, id string, m Movie) error
	Delete(ctx context.Context, id string) error
}

type movieService struct {
	C *mongo.Collection
}

var _ MovieService = (*movieService)(nil)

func NewMovieService(collection *mongo.Collection) MovieService {
	// up to you:
	// indexOpts := new(options.IndexOptions)
	// indexOpts.SetName("movieIndex").
	// 	SetUnique(true).
	// 	SetBackground(true).
	// 	SetSparse(true)

	// collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
	// 	Keys:    []string{"_id", "name"},
	// 	Options: indexOpts,
	// })

	return &movieService{C: collection}
}

func (s *movieService) GetAll(ctx context.Context) ([]Movie, error) {
	// Note:
	// The mongodb's go-driver's docs says that you can pass `nil` to "find all" but this gives NilDocument error,
	// probably it's a bug or a documentation's mistake, you have to pass `bson.D{}` instead.
	cur, err := s.C.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var results []Movie

	for cur.Next(ctx) {
		if err = cur.Err(); err != nil {
			return nil, err
		}

		//	elem := bson.D{}
		var elem Movie
		err = cur.Decode(&elem)
		if err != nil {
			return nil, err
		}

		// results = append(results, Movie{ID: elem[0].Value.(primitive.ObjectID)})

		results = append(results, elem)
	}

	return results, nil
}

func matchID(id string) (bson.D, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	filter := bson.D{{Key: "_id", Value: objectID}}
	return filter, nil
}

var ErrNotFound = errors.New("not found")

func (s *movieService) GetByID(ctx context.Context, id string) (Movie, error) {
	var movie Movie
	filter, err := matchID(id)
	if err != nil {
		return movie, err
	}

	err = s.C.FindOne(ctx, filter).Decode(&movie)
	if err == mongo.ErrNoDocuments {
		return movie, ErrNotFound
	}
	return movie, err
}

func (s *movieService) Create(ctx context.Context, m *Movie) error {
	if m.ID.IsZero() {
		m.ID = primitive.NewObjectID()
	}

	_, err := s.C.InsertOne(ctx, m)
	if err != nil {
		return err
	}

	// The following doesn't work if you have the `bson:"_id` on Movie.ID field,
	// therefore we manually generate a new ID (look above).
	// res, err := ...InsertOne
	// objectID := res.InsertedID.(primitive.ObjectID)
	// m.ID = objectID
	return nil
}

func (s *movieService) Update(ctx context.Context, id string, m Movie) error {
	filter, err := matchID(id)
	if err != nil {
		return err
	}

	// update := bson.D{
	// 	{Key: "$set", Value: m},
	// }
	// ^ this will override all fields, you can do that, depending on your design. but let's check each field:
	elem := bson.D{}

	if m.Name != "" {
		elem = append(elem, bson.E{Key: "name", Value: m.Name})
	}

	if m.Description != "" {
		elem = append(elem, bson.E{Key: "description", Value: m.Description})
	}

	if m.Cover != "" {
		elem = append(elem, bson.E{Key: "cover", Value: m.Cover})
	}

	update := bson.D{
		{Key: "$set", Value: elem},
	}

	_, err = s.C.UpdateOne(ctx, filter, update)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return ErrNotFound
		}
		return err
	}

	return nil
}

func (s *movieService) Delete(ctx context.Context, id string) error {
	filter, err := matchID(id)
	if err != nil {
		return err
	}
	_, err = s.C.DeleteOne(ctx, filter)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return ErrNotFound
		}
		return err
	}

	return nil
}
