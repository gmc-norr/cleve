package gin

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
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

func AddPanelHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.ContentType() != "application/json" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unsupported content type: %s", c.ContentType())})
			return
		}

		var p cleve.GenePanel
		if err := c.ShouldBindJSON(&p); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error(), "when": "parsing panel"})
			return
		}
		if p.Date.IsZero() {
			p.Date = cleve.Time{Time: time.Now()}
		}
		slog.Info("adding panel", "panel", p)
		if err := p.Validate(); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := db.CreatePanel(p); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "a panel with this id and version already exists"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "successfully created the panel", "id": p.Id, "name": p.Name, "version": p.Version})
	}
}
