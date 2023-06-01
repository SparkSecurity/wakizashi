package handler

import (
	"context"
	"github.com/SparkSecurity/wakizashi/manager/db"
	"github.com/SparkSecurity/wakizashi/manager/model"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"time"
)

type CreatePagesRequest struct {
	Urls []string `json:"urls" binding:"required"`
}

// CreatePages is used for appending pages onto a task
func CreatePages(c *gin.Context) {
	// Parse PUT request as json
	var request CreatePagesRequest
	err := c.ShouldBindJSON(&request)
	if err != nil {
		c.AbortWithStatus(400)
		return
	}

	// Get the task from context
	var task model.Task
	taskC, _ := c.Get("task")
	task = taskC.(model.Task)

	// 10 seconds to modify db
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Make an array to store pages to add to db
	pages := make([]interface{}, len(request.Urls))
	for i, url := range request.Urls {
		pages[i] = model.Page{
			ID:     primitive.NewObjectID(),
			TaskID: task.ID,
			Url:    url,
			Status: model.PAGE_STATUS_PENDING_SCRAPE,
		}
	}

	// Insert the pages into the db
	session, err := db.MongoClient.StartSession()
	if err != nil {
		c.AbortWithStatus(500)
		log.Println(err)
		return
	}
	defer session.EndSession(ctx)
	pageRes, err := session.WithTransaction(ctx, func(ctx mongo.SessionContext) (interface{}, error) {
		return db.DB.Collection("page").InsertMany(ctx, pages)
	})
	if err != nil {
		c.AbortWithStatus(500)
		log.Println(err)
		return
	}
	for i, pageID := range pageRes.(*mongo.InsertManyResult).InsertedIDs {
		err := db.PublishScrapeTask(db.ScrapeTask{
			ID:  pageID.(primitive.ObjectID).Hex(),
			Url: request.Urls[i],
		})
		if err != nil {
			log.Println(err)
		}
	}
	c.Status(200)
}

// UpdatePageStatus updates the db of pages
func UpdatePageStatus(success bool, task db.ScrapeTask) {
	// 10 seconds to modify db
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set the status of the page
	newStatus := model.PAGE_STATUS_SCRAPE_SUCCESS
	if !success {
		newStatus = model.PAGE_STATUS_SCRAPE_FAILED
	}

	// Find object id in order to search
	oid, err := primitive.ObjectIDFromHex(task.ID)
	if err != nil {
		log.Println(err)
		return
	}

	// Update the page in the db
	_, err = db.DB.Collection("page").UpdateByID(
		ctx,
		oid,
		bson.D{{"$set", bson.D{
			{"status", newStatus},
			{"response", task.Response},
			{"error", task.Error},
		}}},
	)
	if err != nil {
		log.Println(err)
	}
}
