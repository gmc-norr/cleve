package gin

import (
	"errors"
	"net/http"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)


type PanelQCGetter interface {
	PanelGeneQc(string) (*cleve.GeneQCs, error)
	PanelGenesQCs(cleve.PanelQcFilter) (cleve.GenesQcResult, error)
}

type PanelQCSetter interface {
	CreatePanelQc(cleve.PanelQc) error
}

func AddPanelQcHandler(db PanelQCSetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check MIME type -> Only accepts JSON
		if c.ContentType() != "application/json" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("unsupported content type: %s", c.ContentType())})
			return
		}
		
		var p struct {
			cleve.PanelQc 
		}

		if err := c.ShouldBindJSON(&p); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error(), "when": "parsing genepanel QC"})
			return
		}
		// Validation
		if err := p.Validate(); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := db.CreatePanelQc(p.PanelQc); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Genepanel QC for this runID already exists"})
				return
			}
			if errors.Is(err, mongo.ErrConflict) {
				c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err.Error(), "run id": p.RunId})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "successfully added GeePanel QC data for", "id": p.RunId})



	}
}