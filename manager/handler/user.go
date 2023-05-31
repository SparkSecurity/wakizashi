package handler

import (
	"context"
	"github.com/SparkSecurity/wakizashi/manager/db"
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

// GetUser is a quick function to cast the user from gin context and return it
func GetUser(c *gin.Context) (user User) {
	userAny, _ := c.Get("user")
	user = userAny.(User)
	return
}
