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
	Name  string   `json:"name" binding:"required"`
	Urls  []string `json:"urls"` // deprecated
	Pages []Page   `json:"pages"`
}

type CreateTaskResponse struct {
	TaskID string `json:"taskID"`
}

// CreateTask godoc
// @Summary Create a new task
// @Description Create a new task with the given urls
// @Accept json
// @Produce json
// @Router /task [post]
// @Param request body handler.CreateTaskRequest true "Create Task Request"
// @Success 200 {object} handler.CreateTaskResponse
// @Security auth
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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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

		if len(request.Urls) == 0 && len(request.Pages) == 0 {
			return res, nil
		}
		// Insert each url into a page, then into the task
		taskID := res.InsertedID.(primitive.ObjectID)
		var pages []interface{}
		if len(request.Urls) > 0 {
			pages = make([]interface{}, len(request.Urls))
			for i, url := range request.Urls {
				pages[i] = model.Page{
					ID:      primitive.NewObjectID(),
					TaskID:  taskID,
					Url:     url,
					Browser: false,
					Note:    "",
					Status:  model.PAGE_STATUS_PENDING_SCRAPE,
				}
			}
		} else {
			pages = make([]interface{}, len(request.Pages))
			for i, page := range request.Pages {
				pages[i] = model.Page{
					ID:      primitive.NewObjectID(),
					TaskID:  taskID,
					Url:     page.Url,
					Browser: page.Browser,
					Note:    page.Note,
					Status:  model.PAGE_STATUS_PENDING_SCRAPE,
				}
			}
		}
		insertedIDs := make([]primitive.ObjectID, len(pages))
		for i := 0; i < len(pages); i += 1000 {
			end := i + 1000
			if end > len(pages) {
				end = len(pages)
			}
			pageRes, err := db.DB.Collection("page").InsertMany(ctx, pages[i:end])
			if err != nil {
				return nil, err
			}
			for j, pageID := range pageRes.InsertedIDs {
				insertedIDs[i+j] = pageID.(primitive.ObjectID)
			}
		}
		if err != nil {
			return nil, err
		}
		for i, pageID := range insertedIDs {
			err := db.PublishScrapeTask(db.ScrapeTask{
				ID:      pageID.Hex(),
				Url:     pages[i].(model.Page).Url,
				Browser: pages[i].(model.Page).Browser,
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

type ListTaskResponse struct {
	ID        string `bson:"_id" json:"id"`
	Name      string `bson:"name" json:"name"`
	UserID    string `bson:"userID" json:"userID"`
	CreatedAt string `bson:"createdAt" json:"createdAt"`
}

// ListTask godoc
// @Summary List tasks
// @Description List all tasks created by the auth token
// @Produce json
// @Router /task [get]
// @Success 200 {array} handler.ListTaskResponse
// @Security auth
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

// DownloadTask godoc
// @Summary Download pages
// @Description Download all pages for a given task at its current state into a zip file. The zip structure:
// @Description .
// @Description ├── data
// @Description │   ├── 44d45f44592c966e3049d15c6e2a50209d52168a55e82d2d31a058735304eea7
// @Description │   ├── 72179dada963ca9f154ea2844b614b40b3ba38c7dd99208aaef2e9fd58cca19e
// @Description │   ├── 7b6a484d04943fac714dadb783e8b0fb67fa1a94938507bde7a27b61682afd60
// @Description │   └── 84bc159725f637822a5fc08e6e6551cc7cc1ce11681e6913f10a88b7fae8eef9
// @Description └── index.json
// @Description index json contains the following structure:
// @Description [
// @Description   {
// @Description     "id": "<page id>",
// @Description     "url": "<page url>",
// @Description     "bodyHash": "<page body hash value>",
// @Description   }...
// @Description ]
// @Produce octet-stream
// @Router /task/{task_id} [get]
// @Param task_id path string true "Task ID"
// @Param indexOnly query string false "true/false: Only download index json"
// @Success 200 "zip file"
// @Security auth
func DownloadTask(c *gin.Context) {
	// Get the task from context
	var task model.Task
	taskC, _ := c.Get("task")
	task = taskC.(model.Task)

	// Get all pages for the task
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
		err := util.ZipFile(successfulPages, w, c.Query("indexOnly") == "true")
		if err != nil {
			log.Println(err)
			return false
		}
		return false
	})
}

// GetStats godoc
// @Summary Get statistics for the specific task
// @Produce json
// @Router /task/{task_id}/statistics [get]
// @Param task_id path string true "Task ID"
// @Success 200 {object} handler.stats
// @Security auth
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
	stats, err := getStatistics(taskID)
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

// getStatistics is a helper function to get statistics for a given task
// accepts a list of pages and returns a statistics struct
func getStatistics(taskID primitive.ObjectID) (*stats, error) {
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
					{"$sum", 1},
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

// DeleteTask godoc
// @Summary Delete specific task
// @Router /task/{task_id} [delete]
// @Param task_id path string true "Task ID"
// @Success 200
// @Security auth
func DeleteTask(c *gin.Context) {
	// Get the task from context
	var task model.Task
	taskC, _ := c.Get("task")
	task = taskC.(model.Task)

	// 10 seconds to modify db
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Insert the pages into the db
	session, err := db.MongoClient.StartSession()
	if err != nil {
		c.AbortWithStatus(500)
		log.Println(err)
		return
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(ctx mongo.SessionContext) (interface{}, error) {
		_, err := db.DB.Collection("task").DeleteOne(ctx, bson.D{
			{"_id", task.ID},
		})
		if err != nil {
			return nil, err
		}
		return db.DB.Collection("page").DeleteMany(ctx, bson.D{
			{"taskID", task.ID},
		})
	})
	if err != nil {
		c.AbortWithStatus(500)
		log.Println(err)
		return
	}
	c.Status(200)
}
