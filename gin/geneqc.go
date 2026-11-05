package gin

import (
	"errors"
	"net/http"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)


type GeneQCGetter interface {
	GeneQc(string) (*cleve.GeneQCs, error)
	GenesQCs(cleve.PanelQcFilter) (cleve.GenesQcResult, error)
	AllGeneQc(string) (*cleve.GeneQc, error)
	AllGeneQCs(cleve.PanelQcFilter) (cleve.AllGenesQcResult, error)
	ExonQc(string,float64,string) (*cleve.ExonQCs, error)
	ExonsQCs(cleve.PanelQcFilter) (cleve.ExonQcResult, error)
}

type GeneQCSetter interface {
	CreateGeneQc(cleve.GenesQc) error
	UpdateGeneQc(string,cleve.GeneQc) (error)
}

type GeneQCStore interface {
    GeneQCGetter
    GeneQCSetter
}


func GenesQCHandler(db GeneQCGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getPanelQcFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		geneqcs, err := db.AllGeneQCs(filter)

		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("GeneQC data not found")})
				return
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, geneqcs.AllGenes)
		
	}
}

func GeneQCHandler(db GeneQCGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		symbol := c.Param("symbol")
		geneqc, err := db.AllGeneQc(symbol)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("GeneQC data for GeneID: %s not found", symbol)})
				return
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, geneqc)
	}
}

func AddGeneQcHandler(db GeneQCSetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		
		var p struct {
			cleve.GenesQc
		}

		
		if err := c.ShouldBindJSON(&p); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error(), "when": "parsing gene QC"})
			return
		}

		// Validation
		if err := p.Validate(); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := db.CreateGeneQc(p.GenesQc); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Gene QC already exists for", "runid": p.RunId})
				return
			}
			if errors.Is(err, mongo.ErrConflict) {
				c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err.Error(), "run id": p.RunId})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "successfully added Gene QC data for", "id": p.RunId})

	}
}

func UpdateGeneQcHandler(db GeneQCStore) gin.HandlerFunc {
	return func(c *gin.Context) {

		runId := c.Param("runId")
		filter := cleve.PanelQcFilter {
		RunId: runId,
		}
		_, err := db.GenesQCs(filter)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.JSON(http.StatusNotFound, gin.H{"error": "run not found", "run_id": runId})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		
		var p struct {
			cleve.GeneQc
		}

		if err := c.ShouldBindJSON(&p); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error(), "when": "parsing gene QC"})
			return
		}

		// Check if data for the Gene already exists
		filter = cleve.PanelQcFilter {
		Symbol: p.Symbol,
		RunId: runId,
		}
		_, err = db.GenesQCs(filter)
		if err != mongo.ErrNoDocuments {
			c.JSON(http.StatusBadRequest, gin.H{"error": "gene QC already exists for runID", "when": "updating gene QC"})
			return
		}

		if err := db.UpdateGeneQc(runId, p.GeneQc); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "failed to update gene QC for run", "run": runId, "error": err})
			return
		}


		

	}
}
