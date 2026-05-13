package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type SavedAnalysis struct {
	ID             bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID         bson.ObjectID `bson:"user_id" json:"userId"`
	Ticker         string        `bson:"ticker" json:"ticker"`
	Recommendation string        `bson:"recommendation" json:"recommendation"`
	Reasoning      []string      `bson:"reasoning" json:"reasoning"`
	Persona        string        `bson:"persona" json:"persona"`
	PriceAtSave    float64       `bson:"price_at_save" json:"priceAtSave"`
	SavedAt        time.Time     `bson:"saved_at" json:"savedAt"`
}
