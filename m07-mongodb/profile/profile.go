// Package profile holds the document domain (Profile + Event) and a MongoDB
// repository. It is shaped to satisfy the M5 profile.Repo gRPC interface.
package profile

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ErrNotFound = errors.New("profile: not found")

// Profile is a flexible bag of traits keyed by customer id (used as _id).
type Profile struct {
	CustomerID string            `bson:"_id"`
	Traits     map[string]string `bson:"traits"`
	UpdatedAt  time.Time         `bson:"updated_at"`
}

// Event is an append-only record in a dedicated collection.
type Event struct {
	ID         string    `bson:"_id"`
	CustomerID string    `bson:"customer_id"`
	Type       string    `bson:"type"`
	OccurredAt time.Time `bson:"occurred_at"`
}

type MongoRepo struct {
	profiles *mongo.Collection
	events   *mongo.Collection
}

func NewMongoRepo(ctx context.Context, client *mongo.Client, dbName string) (*MongoRepo, error) {
	db := client.Database(dbName)
	r := &MongoRepo{
		profiles: db.Collection("profiles"),
		events:   db.Collection("events"),
	}
	if err := r.ensureIndexes(ctx); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *MongoRepo) ensureIndexes(ctx context.Context) error {
	_, err := r.events.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "customer_id", Value: 1},
			{Key: "occurred_at", Value: -1},
		},
		Options: options.Index().SetName("by_customer_time"),
	})
	return err
}

func (r *MongoRepo) Upsert(ctx context.Context, p Profile) (Profile, error) {
	p.UpdatedAt = time.Now().UTC()
	_, err := r.profiles.UpdateOne(ctx,
		bson.M{"_id": p.CustomerID},
		bson.M{"$set": bson.M{"traits": p.Traits, "updated_at": p.UpdatedAt}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return Profile{}, fmt.Errorf("upsert profile: %w", err)
	}
	return p, nil
}

func (r *MongoRepo) Get(ctx context.Context, customerID string) (Profile, error) {
	var p Profile
	err := r.profiles.FindOne(ctx, bson.M{"_id": customerID}).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("get profile: %w", err)
	}
	return p, nil
}

func (r *MongoRepo) AppendEvent(ctx context.Context, e Event) error {
	if _, err := r.events.InsertOne(ctx, e); err != nil {
		return fmt.Errorf("append event: %w", err)
	}
	return nil
}

func (r *MongoRepo) Events(ctx context.Context, customerID string) ([]Event, error) {
	cur, err := r.events.Find(ctx,
		bson.M{"customer_id": customerID},
		options.Find().SetSort(bson.D{{Key: "occurred_at", Value: 1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("find events: %w", err)
	}
	defer cur.Close(ctx)
	var out []Event
	if err := cur.All(ctx, &out); err != nil {
		return nil, fmt.Errorf("decode events: %w", err)
	}
	return out, nil
}

// EventCountsByType is an aggregation: number of events per type for a customer.
func (r *MongoRepo) EventCountsByType(ctx context.Context, customerID string) (map[string]int, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"customer_id": customerID}}},
		{{Key: "$group", Value: bson.M{"_id": "$type", "count": bson.M{"$sum": 1}}}},
	}
	cur, err := r.events.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("aggregate: %w", err)
	}
	defer cur.Close(ctx)

	var rows []struct {
		Type  string `bson:"_id"`
		Count int    `bson:"count"`
	}
	if err := cur.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("decode agg: %w", err)
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.Type] = row.Count
	}
	return out, nil
}
