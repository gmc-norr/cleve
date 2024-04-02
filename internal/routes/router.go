package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve/internal/db"
	"net/http"
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

		_, err := db.GetKey(requestKey)
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

func NewRouter() http.Handler {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		IndexHandler(c, r.Routes())
	})

	r.GET("/api/runs", RunsHandler)
	r.GET("/api/runs/:runId", RunHandler)

	authEndpoints := r.Group("/")
	authEndpoints.Use(authMiddleware())
	authEndpoints.POST("/api/runs", AddRunHandler)
	authEndpoints.PATCH("/api/runs/:runId", UpdateRunHandler)

	return r
}

func IndexHandler(c *gin.Context, routes []gin.RouteInfo) {
	c.HTML(http.StatusOK, "index.tmpl", gin.H{"routes": routes})
}
