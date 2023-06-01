package util

import (
	"github.com/SparkSecurity/wakizashi/manager/db"
	"github.com/SparkSecurity/wakizashi/manager/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type stats struct {
	TotalPages int64 `bson:"total"`
	Successful int64 `bson:"successful"`
	Failed     int64 `bson:"failed"`
	InProgress int64 `bson:"inProgress"`
}

type aggregateResult struct {
	Status int64 `bson:"status"`
	Count  int64 `bson:"count"`
}

// GetStatistics is a helper function to get statistics for a given task
// accepts a list of pages and returns a statistics struct
func GetStatistics(taskID primitive.ObjectID) (*stats, error) {
	ctx, cancel := TimeoutContext(5 * time.Second)
	defer cancel()

	// Search db for all pages related to the task
	cursor, err := db.DB.Collection("page").Aggregate(ctx, bson.A{
		bson.D{
			{"$match", bson.D{
				{"taskID", taskID},
			}},
		},
		bson.D{
			{"$group", bson.D{
				{"status", "$status"},
				{"count", bson.D{
					{"$count", bson.D{}},
				}},
			}},
		},
	})
	if err != nil {
		return nil, err
	}

	// Convert the cursor to a list of aggregate results
	var dbResult []aggregateResult
	err = cursor.Decode(&dbResult)
	if err != nil {
		return nil, err
	}

	// Count
	var result stats
	for _, r := range dbResult {
		switch r.Status {
		case model.PAGE_STATUS_PENDING_SCRAPE:
			result.InProgress = r.Count
		case model.PAGE_STATUS_SCRAPE_SUCCESS:
			result.Successful = r.Count
		case model.PAGE_STATUS_SCRAPE_FAILED:
			result.Failed = r.Count
		}
		result.TotalPages += r.Count
	}

	return &result, nil
}
