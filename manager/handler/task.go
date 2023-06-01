package handler

import (
	"context"
	"github.com/SparkSecurity/wakizashi/manager/db"
	"github.com/SparkSecurity/wakizashi/manager/model"
	"github.com/SparkSecurity/wakizashi/manager/util"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"io"
	"log"
	"time"
)

type CreateTaskRequest struct {
	Name string   `json:"name" binding:"required"`
	Urls []string `json:"urls"`
}

func CreateTask(c *gin.Context) {
	// converting request to struct
	var request CreateTaskRequest
	err := c.ShouldBindJSON(&request)
	if err != nil {
		c.AbortWithStatus(400)
		return
	}

	user := GetUser(c)

	// 10 second to create the task
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	session, err := db.MongoClient.StartSession()
	if err != nil {
		c.AbortWithStatus(500)
		log.Println(err)
		return
	}
	defer session.EndSession(ctx)

	// Begin transaction for inserting task
	res, err := session.WithTransaction(ctx, func(ctx mongo.SessionContext) (interface{}, error) {
		// Creates and inserts a new task
		res, err := db.DB.Collection("task").InsertOne(ctx, model.Task{
			ID:        primitive.NewObjectID(),
			Name:      request.Name,
			UserID:    user.ID,
			CreatedAt: primitive.DateTime(time.Now().UnixMilli()),
		})
		if err != nil {
			return nil, err
		}

		// Insert each url into a page, then into the task
		taskID := res.InsertedID.(primitive.ObjectID)
		pages := make([]interface{}, len(request.Urls))
		for i, url := range request.Urls {
			pages[i] = model.Page{
				ID:     primitive.NewObjectID(),
				TaskID: taskID,
				Url:    url,
				Status: model.PAGE_STATUS_PENDING_SCRAPE,
			}
		}
		pageRes, err := db.DB.Collection("page").InsertMany(ctx, pages) // Actually insert the pages
		if err != nil {
			return nil, err
		}
		for i, pageID := range pageRes.InsertedIDs {
			err := db.PublishScrapeTask(db.ScrapeTask{
				ID:  pageID.(primitive.ObjectID).Hex(),
				Url: request.Urls[i],
			})
			if err != nil {
				log.Println(err)
			}
		}
		return res, nil
	})
	if err != nil {
		c.AbortWithStatus(500)
		log.Println(err)
		return
	}

	// Get the task id in hex and bop it back
	taskID := res.(*mongo.InsertOneResult).InsertedID.(primitive.ObjectID).Hex()
	c.JSON(200, gin.H{
		"taskID": taskID,
	})
}

// ListTask lists all tasks by a given user
func ListTask(c *gin.Context) {
	// Get the user and find tasks with their user id
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// converting interface into user struct
	user := GetUser(c)
	cursor, err := db.DB.Collection("task").Find(ctx, bson.D{
		{"userID", user.ID},
	})
	if err != nil {
		c.AbortWithStatus(500)
		log.Println(err)
		return
	}

	// Creates a tasks array for serializing into json
	var tasks []model.Task
	err = cursor.All(ctx, &tasks)
	if err != nil {
		c.AbortWithStatus(500)
		log.Println(err)
		return
	}
	c.JSON(200, tasks)
}

// DownloadTask downloads all pages for a given task at its current state
func DownloadTask(c *gin.Context) {
	// Get the task from context
	var task model.Task
	taskC, _ := c.Get("task")
	task = taskC.(model.Task)

	// Get all pages for the task
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.DB.Collection("page").Find(ctx, bson.D{
		{"taskID", task.ID},
	})
	if err != nil {
		c.AbortWithStatus(500)
		log.Println(err)
		return
	}

	var pages []model.Page
	err = cursor.All(ctx, &pages)
	if err != nil {
		c.AbortWithStatus(500)
		log.Println(err)
		return
	}

	// Loop through each page to find completed ones
	successfulPages := make([]model.Page, 0)
	for _, page := range pages {
		if page.Status != model.PAGE_STATUS_SCRAPE_SUCCESS {
			continue
		}

		successfulPages = append(successfulPages, page)
	}

	// zip the pages and send it back
	c.Writer.Header().Set("Content-type", "application/octet-stream")
	c.Writer.Header().Set("Content-Disposition", `attachment; filename=`+task.Name+`.zip`)
	c.Stream(func(w io.Writer) bool {
		err := util.ZipFile(successfulPages, w)
		if err != nil {
			log.Println(err)
			return false
		}

		return false
	})
}

func GetStats(c *gin.Context) {
	// Get the task from context
	var task model.Task
	taskC, _ := c.Get("task")
	task = taskC.(model.Task)

	// Get the object ID of the task
	taskID, err := primitive.ObjectIDFromHex(task.ID.Hex())
	if err != nil {
		c.AbortWithStatus(500)
		log.Println(err)
		return
	}

	// Get the statistics
	stats, err := GetStatistics(taskID)
	if err != nil {
		c.AbortWithStatus(500)
		log.Println(err)
		return
	}

	// Serialize the statistics
	c.JSON(200, stats)
}

type stats struct {
	TotalPages int64 `json:"total"`
	Successful int64 `json:"successful"`
	Failed     int64 `json:"failed"`
	InProgress int64 `json:"inProgress"`
}

type aggregateResult struct {
	Status int64 `bson:"_id"`
	Count  int64 `bson:"count"`
}

// GetStatistics is a helper function to get statistics for a given task
// accepts a list of pages and returns a statistics struct
func GetStatistics(taskID primitive.ObjectID) (*stats, error) {
	ctx, cancel := util.TimeoutContext(5 * time.Second)
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
				{"_id", "$status"},
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
	err = cursor.All(ctx, &dbResult)
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
