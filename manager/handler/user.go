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

type User struct {
	ID    primitive.ObjectID `bson:"_id"`
	Token string             `bson:"token"`
}

// AuthMiddleware Check if user exists in db
func AuthMiddleware(c *gin.Context) {
	// Kill if no user token
	token := c.GetHeader("token")
	if token == "" {
		// TODO: return some error message?
		c.AbortWithStatus(401)
		return
	}

	// 5 second search of DB for the user from token
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := db.DB.Collection("user").FindOne(ctx, bson.D{
		{"token", token},
	})
	if result.Err() == mongo.ErrNoDocuments {
		c.AbortWithStatus(401)
		return
	}

	// Pass the user details into gin context
	var user User
	err := result.Decode(&user)
	if err != nil {
		c.AbortWithStatus(500)
		log.Println(err)
		return
	}

	c.Set("user", user)
	c.Next()
}

// AuthMiddlewareGetTask checks if the user owns the task
// this function assumes that the user is already authenticated
// it sets the task in the context
func AuthMiddlewareGetTask(c *gin.Context) {
	type TaskID struct {
		ID string `uri:"id" binding:"required"`
	}

	var taskID TaskID
	err := c.ShouldBindUri(&taskID)
	if err != nil {
		c.AbortWithStatus(400)
		return
	}

	// 5 second search of DB for the user from token
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get the user from the context
	user := GetUser(c)

	// Get the task from the db
	taskIDObj, err := primitive.ObjectIDFromHex(taskID.ID)
	if err != nil {
		c.AbortWithStatus(400)
		return
	}

	result := db.DB.Collection("task").FindOne(ctx, bson.D{
		{"_id", taskIDObj},
	})
	if result.Err() == mongo.ErrNoDocuments {
		c.JSON(400, gin.H{
			"error": "task not found",
		})
		c.Abort()
		return
	}
	// Check if the user owns the task
	var task model.Task
	err = result.Decode(&task)
	if err != nil {
		c.AbortWithStatus(500)
		log.Println(err)
		return
	}

	if task.UserID != user.ID {
		c.AbortWithStatus(403)
		return
	}

	c.Set("task", task)
	c.Next()
}

// GetUser is a quick function to cast the user from gin context and return it
func GetUser(c *gin.Context) (user User) {
	userAny, _ := c.Get("user")
	user = userAny.(User)
	return
}
