package gin

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/viper"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

func multiply(x, y float64) float64 {
	return x * y
}

func title(s string) string {
	return cases.Title(language.English).String(s)
}

func toFloat(x interface{}) float64 {
	switch v := x.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	}
	return 0
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
	r.SetFuncMap(template.FuncMap{
		"multiply": multiply,
		"title":    title,
		"toFloat":  toFloat,
	})
	r.LoadHTMLGlob(fmt.Sprintf("%s/templates/*", viper.GetString("assets")))

	// Dashboard endpoints
	r.GET("/", DashboardHandler(db))
	r.GET("/runs", DashboardHandler(db))
	r.GET("/runs/:runId", DashboardRunHandler(db))
	r.GET("/qc", DashboardQCHandler(db))
	r.GET("/qc/charts/global", GlobalChartsHandler(db))

	// API endpoints
	r.GET("/api", func(c *gin.Context) {
		ApiIndexHandler(c, r.Routes())
	})

	r.GET("/api/runs", RunsHandler(db))
	r.GET("/api/runs/:runId", RunHandler(db))
	r.GET("/api/runs/:runId/analysis", AnalysesHandler(db))
	r.GET("/api/runs/:runId/analysis/:analysisId", AnalysisHandler(db))
	r.GET("/api/runs/:runId/qc", RunQcHandler(db))
	r.GET("/api/platforms", PlatformsHandler(db))
	r.GET("/api/platforms/:platformName", GetPlatformHandler(db))
	r.GET("/api/qc/:platformName", AllQcHandler(db))

	authEndpoints := r.Group("/")
	authEndpoints.Use(authMiddleware(db))
	authEndpoints.POST("/api/runs", AddRunHandler(db))
	authEndpoints.PATCH("/api/runs/:runId", UpdateRunHandler(db))
	authEndpoints.POST("/api/runs/:runId/analysis", AddAnalysisHandler(db))
	authEndpoints.PATCH("/api/runs/:runId/analysis/:analysisId", UpdateAnalysisHandler(db))
	authEndpoints.POST("/api/runs/:runId/qc", AddRunQcHandler(db))
	authEndpoints.POST("/api/platforms", AddPlatformHandler(db))

	r.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "error404", nil)
	})

	return r
}

func ApiIndexHandler(c *gin.Context, routes []gin.RouteInfo) {
	c.HTML(http.StatusOK, "api.tmpl", gin.H{"routes": routes})
}
