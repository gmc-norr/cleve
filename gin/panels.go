package gin

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve/mongo"
)

func PanelsHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getPanelFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": err},
			)
			return
		}
		slog.Info("api panels", "filter", filter)
		panels, err := db.Panels(filter)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.JSON(http.StatusOK, panels)
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err})
			return
		}
		c.JSON(http.StatusOK, panels)
	}
}

func PanelHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		panelId := c.Param("panelId")
		filter, err := getPanelFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		panel, err := db.Panel(panelId, filter.Version)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
					"error":   "panel not found",
					"id":      panelId,
					"version": filter.Version,
				})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, panel)
	}
}
