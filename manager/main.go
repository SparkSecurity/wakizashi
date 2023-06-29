package main

import (
	"fmt"
	"github.com/SparkSecurity/wakizashi/manager/config"
	"github.com/SparkSecurity/wakizashi/manager/db"
	docs "github.com/SparkSecurity/wakizashi/manager/docs"
	"github.com/SparkSecurity/wakizashi/manager/handler"
	"github.com/SparkSecurity/wakizashi/manager/storage"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Wakizashi Scrape Platform Manager API
// @securityDefinitions.apikey auth
// @in header
// @name token
func main() {
	docs.SwaggerInfo.InstanceName()
	// Init the DB things
	config.LoadConfig()
	storage.CreateStorage()
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
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	// For user based tasks
	authRequired := router.Group("")
	authRequired.Use(handler.AuthMiddleware)

	authRequired.POST("/task", handler.CreateTask)
	authRequired.GET("/task", handler.ListTask)

	// For task based tasks
	taskRequired := authRequired.Group("")
	taskRequired.Use(handler.AuthMiddlewareGetTask)

	taskRequired.GET("/task/:id", handler.DownloadTask)
	taskRequired.DELETE("/task/:id", handler.DeleteTask)
	taskRequired.PUT("/task/:id", handler.CreatePages)
	taskRequired.GET("/task/:id/statistics", handler.GetStats)

	// Start
	err := router.Run(fmt.Sprintf(":%d", config.Config.ListenPort))
	if err != nil {
		panic(err)
	}
}
