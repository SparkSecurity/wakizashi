package main

import (
	"fmt"
	"github.com/SparkSecurity/wakizashi/manager/config"
	"github.com/SparkSecurity/wakizashi/manager/db"
	"github.com/SparkSecurity/wakizashi/manager/handler"
	"github.com/gin-gonic/gin"
)

func main() {
	// Init the DB things
	config.LoadConfig()
	db.DBConnect()
	defer db.DBDisconnect()
	db.MQConnect()
	defer db.MQDisconnect()
	go db.MQConsumeResponse(handler.UpdatePageStatus)
	// Setting our API points
	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		c.String(418, "meow-tea")
	})

	// Setting up authentication
	authRequired := router.Group("")
	authRequired.Use(handler.AuthMiddleware)

	authRequired.POST("/task", handler.CreateTask)
	authRequired.GET("/task", handler.ListTask)
	authRequired.PUT("/task", handler.CreatePages)

	// Start
	err := router.Run(fmt.Sprintf(":%d", config.Config.ListenPort))
	if err != nil {
		panic(err)
	}
}
