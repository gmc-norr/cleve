package gin

import (
	"errors"
	"net/http"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)


type PanelQCGetter interface {
	PanelQc(string) (*cleve.PanelQc, error)
	PanelQCs(cleve.PanelQcFilter) (cleve.PanelQcResult, error)
	GeneQc(string) (*cleve.GeneQCs, error)
	GenesQCs(cleve.PanelQcFilter) (cleve.GeneQcResult, error)
	ExonQc(float64) (*cleve.ExonQCs, error)
	ExonsQCs(cleve.PanelQcFilter) (cleve.ExonQcResult, error)
}

type PanelQCSetter interface {
	CreatePanelQc(cleve.PanelQc) error

}

func PanelQCsHandler(db PanelQCGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getPanelQcFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		panelqcs, err := db.PanelQCs(filter)

		if errors.As(err, &mongo.PageOutOfBoundsError{}) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, panelqcs)
	}
}

func PanelQCHandler(db PanelQCGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		runId := c.Param("runId")
		panelqc, err := db.PanelQc(runId)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.JSON(http.StatusNotFound, gin.H{"error": "panelqc not found"})
				return
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, panelqc)
	}
}

func GenesQCHandler(db PanelQCGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getPanelQcFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		panelqcs, err := db.GenesQCs(filter)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("GeneQC data not found")})
				return
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, panelqcs.GeneQcs)
		
	}
} 

func GeneQCHandler(db PanelQCGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		hgnc := c.Param("hgnc")
		geneqc, err := db.GeneQc(hgnc)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("GeneQC data for GeneID: %s not found", hgnc)})
				return
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, geneqc)
	}
}

func ExonsQCHandler(db PanelQCGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getPanelQcFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		panelqcs, err := db.ExonsQCs(filter)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("ExonQC data not found")})
				return
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, panelqcs.ExonQcs)
		
	}
}

func ExonQCHandler(db PanelQCGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		exon_number := c.Param("exon_number")
		exon_number_float, err := strconv.ParseFloat(exon_number, 64)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Cannot convert %s to float", exon_number)})
			return
		}
		exonqc, err := db.ExonQc(exon_number_float)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("ExonQC data for Exon number: %s not found", exon_number)})
				return
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, exonqc)
	}
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