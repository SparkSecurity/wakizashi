package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Task struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	Name      string             `bson:"name" json:"name"`
	UserID    primitive.ObjectID `bson:"userID" json:"userID"`
	CreatedAt primitive.DateTime `bson:"createdAt" json:"createdAt"`
}
