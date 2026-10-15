package gin

import (
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
	ExonQc(string,float64) (*cleve.ExonQCs, error)
	ExonsQCs(cleve.PanelQcFilter) (cleve.ExonQcResult, error)
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