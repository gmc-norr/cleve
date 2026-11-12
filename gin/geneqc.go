package gin

import (
	"errors"
	"net/http"
	"fmt"
	"os"
	"io"
	"encoding/json"

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
		
		if errors.As(err, &mongo.PageOutOfBoundsError{}) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err == nil && len(geneqcs.AllGenes) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("GeneQC data not found")})
		} else {
			c.JSON(http.StatusOK, geneqcs)
		}

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
		
		var updateRequest struct {
			Path  string              `json:"path"`
		}
		if err := c.BindJSON(&updateRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		path, err := os.Stat(updateRequest.Path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
					"error":       "Path does not exists",
					"path": updateRequest.Path,
				})
				return
			}
			if errors.Is(err, os.ErrPermission) {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error":       "Permission denied",
					"path": updateRequest.Path,
				})
				return
			}
		}
		if path.IsDir() {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error":   "path is a directory",
				"path": updateRequest.Path,
			})
			return
		}

		f, err := os.Open(updateRequest.Path)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to open file", "when": "adding gene QC"})
			return
		}

		defer func() {
			if err := f.Close(); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to close file", "when": "adding gene QC"})
			return
			}
		}()

		data, err := io.ReadAll(f)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Could not read file", "when": "adding gene QC"})
			return
		}

		var geneqc cleve.GenesQc
		err = json.Unmarshal([]byte(data), &geneqc)

		if err := geneqc.Validate(); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := db.CreateGeneQc(geneqc); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Gene QC already exists for", "runid": geneqc.RunId})
				return
			}
			if errors.Is(err, mongo.ErrConflict) {
				c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err.Error(), "run id": geneqc.RunId})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "successfully added Gene QC data for", "id": geneqc.RunId})
	}
}

func UpdateGeneQcHandler(db GeneQCStore) gin.HandlerFunc {
	return func(c *gin.Context) {


		var updateRequest struct {
			Files  cleve.GeneQCFiles    `json:"files"`
		}
		if err := c.BindJSON(&updateRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		runId := c.Param("runId")
		filter := cleve.PanelQcFilter {
		RunId: runId,
		}
		_, err := db.GenesQCs(filter)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
					"error":       "RunID not found",
					"RunID": runId,
				})
				return
			}
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error":   "failed to fetch runID",
				"details": err,
			})
			return
		}
		
		if len(updateRequest.Files) > 0 {
			for i := range updateRequest.Files {
				path, err := os.Stat(updateRequest.Files[i].Path)
				if err != nil {
					if errors.Is(err, os.ErrNotExist) {
						c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
							"error":       "Path does not exists",
							"path": updateRequest.Files[i].Path,
						})
						return
					}
					if errors.Is(err, os.ErrPermission) {
						c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
							"error":       "Permission denied",
							"path": updateRequest.Files[i].Path,
						})
						return
					}
				}
				if path.IsDir() {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
						"error":   "path is a directory",
						"path": updateRequest.Files[i].Path,
					})
					return
				}

				f, err := os.Open(updateRequest.Files[i].Path)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to open file", "when": "updating gene QC"})
					return
				}

				defer func() {
					if err := f.Close(); err != nil {
						c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to close file", "when": "updating gene QC"})
					return
					}
				}()

				data, err := io.ReadAll(f)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Could not read file", "when": "updating gene QC"})
					return
				}

				var geneqc cleve.GeneQc
				err = json.Unmarshal([]byte(data), &geneqc)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "failed to Unmarshal JSON file", "when": "updating gene QC"})
					return
				}

				if err := geneqc.ValidateUpdate(); err != nil {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				// Check if data for the Gene already exists
				filter = cleve.PanelQcFilter {
				Symbol: geneqc.Symbol,
				RunId: runId,
				}
				_, err = db.GenesQCs(filter)
				if err != mongo.ErrNoDocuments {
					c.JSON(http.StatusBadRequest, gin.H{"error": "gene QC already exists for runID", "when": "updating gene QC", "symbol": geneqc.Symbol})
					return
				}

				if err := db.UpdateGeneQc(runId, geneqc); err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "failed to update gene QC for run", "run": runId, "error": err})
					return
				}

				c.JSON(http.StatusOK, gin.H{"message": "successfully updated Gene QC data for", "id": runId, "symbol": geneqc.Symbol})

			}
			
		}

	}
}