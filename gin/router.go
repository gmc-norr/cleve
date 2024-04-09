package gin

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

func authMiddleware(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestKey := c.Request.Header.Get("Authorization")
		if requestKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "missing authorization header",
			})
			return
		}

		_, err := db.Keys.Get(requestKey)
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

func NewRouter(db *mongo.DB, debug bool) http.Handler {
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

	r.GET("/api/runs", RunsHandler(db))
	r.GET("/api/runs/:runId", RunHandler(db))
	r.GET("/api/runs/:runId/analysis", AnalysesHandler(db))
	r.GET("/api/runs/:runId/analysis/:analysisId", AnalysisHandler(db))

	authEndpoints := r.Group("/")
	authEndpoints.Use(authMiddleware(db))
	authEndpoints.POST("/api/runs", AddRunHandler(db))
	authEndpoints.PATCH("/api/runs/:runId", UpdateRunHandler(db))
	authEndpoints.POST("/api/runs/:runId/analysis", AddAnalysisHandler(db))
	authEndpoints.PATCH("/api/runs/:runId/analysis/:analysisId", UpdateAnalysisHandler(db))

	return r
}

func IndexHandler(c *gin.Context, routes []gin.RouteInfo) {
	c.HTML(http.StatusOK, "index.tmpl", gin.H{"routes": routes})
}
