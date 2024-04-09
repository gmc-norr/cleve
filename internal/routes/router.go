package routes

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/viper"
	"io"
	"log"
	"net/http"
	"os"
)

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestKey := c.Request.Header.Get("Authorization")
		if requestKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "missing authorization header",
			})
			return
		}

		_, err := mongo.GetKey(requestKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid API key",
			})
			return
		}

		c.Next()
	}
}

func NewRouter(debug bool) http.Handler {
	gin.DisableConsoleColor()
	if viper.GetString("logfile") != "" {
		f, err := os.Create(viper.GetString("logfile"))
		if err != nil {
			log.Fatal(err)
		}
		gin.DefaultWriter = io.MultiWriter(f, os.Stdout)
	}

	if debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.LoadHTMLGlob(fmt.Sprintf("%s/templates/*", viper.GetString("assets")))

	r.GET("/", func(c *gin.Context) {
		IndexHandler(c, r.Routes())
	})

	r.GET("/api/runs", RunsHandler)
	r.GET("/api/runs/:runId", RunHandler)
	r.GET("/api/runs/:runId/analysis", AnalysesHandler)
	r.GET("/api/runs/:runId/analysis/:analysisId", AnalysisHandler)

	authEndpoints := r.Group("/")
	authEndpoints.Use(authMiddleware())
	authEndpoints.POST("/api/runs", AddRunHandler)
	authEndpoints.PATCH("/api/runs/:runId", UpdateRunHandler)
	authEndpoints.POST("/api/runs/:runId/analysis", AddAnalysisHandler)
	authEndpoints.PATCH("/api/runs/:runId/analysis/:analysisId", UpdateAnalysisHandler)

	return r
}

func IndexHandler(c *gin.Context, routes []gin.RouteInfo) {
	c.HTML(http.StatusOK, "index.tmpl", gin.H{"routes": routes})
}
