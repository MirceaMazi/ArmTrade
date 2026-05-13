package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type WatchlistItem struct {
	ID       bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID   bson.ObjectID `bson:"user_id" json:"userId"`
	Ticker   string        `bson:"ticker" json:"ticker"`
	AddedAt  time.Time     `bson:"added_at" json:"addedAt"`
	BuyPrice *float64      `bson:"buy_price,omitempty" json:"buyPrice,omitempty"`
	Quantity *float64      `bson:"quantity,omitempty" json:"quantity,omitempty"`
	BuyDate  *time.Time    `bson:"buy_date,omitempty" json:"buyDate,omitempty"`
}
