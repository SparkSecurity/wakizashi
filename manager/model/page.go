package model

import "go.mongodb.org/mongo-driver/bson/primitive"

const (
	PAGE_STATUS_PENDING_SCRAPE = 0
	PAGE_STATUS_SCRAPE_SUCCESS = 1
	PAGE_STATUS_SCRAPE_FAILED  = 2
)

type Page struct {
	ID       primitive.ObjectID `bson:"_id" json:"id"`
	TaskID   primitive.ObjectID `bson:"taskID" json:"taskID"`
	Url      string             `bson:"url" json:"url"`
	Browser  bool               `bson:"browser" json:"browser"`
	Note     string             `bson:"note" json:"note"`
	Status   int                `bson:"status" json:"status"`
	Response string             `bson:"response" json:"response"`
	Error    []string           `bson:"error" json:"error"`
}
