package routes

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func NewRouter() http.Handler {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		IndexHandler(c, r.Routes())
	})

	r.GET("/api/runs", RunsHandler)
	r.GET("/api/runs/:runId", RunHandler)
	r.POST("/api/runs", AddRunHandler)
	r.PATCH("/api/runs/:runId", UpdateRunHandler)

	return r
}

func IndexHandler(c *gin.Context, routes []gin.RouteInfo) {
	c.HTML(http.StatusOK, "index.tmpl", gin.H{"routes": routes})
}
