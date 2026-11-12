package gin

import (
	"errors"
	"net/http"
	"os"
	"io"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)


type PanelQCGetter interface {
	PanelGeneQc(string, string) (*cleve.GeneQCs, error)
	PanelGenesQCs(cleve.PanelQcFilter) (cleve.GenesQcResult, error)
}

type PanelQCSetter interface {
	CreatePanelQc(cleve.PanelQc) error
}

func AddPanelQcHandler(db PanelQCSetter) gin.HandlerFunc {
	return func(c *gin.Context) {

		var updateRequest struct {
			Path    string              `json:"path"`
		}
		if err := c.BindJSON(&updateRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if updateRequest.Path != "" {

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
				c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to open file", "when": "adding panelgene QC"})
				return
			}

			defer func() {
				if err := f.Close(); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to close file", "when": "adding panelgene QC"})
				return
				}
			}()

			data, err := io.ReadAll(f)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Could not read file", "when": "adding panelgene QC"})
				return
			}

			var panelqc cleve.PanelQc

			err = json.Unmarshal([]byte(data), &panelqc)
			// Validation
			if err := panelqc.Validate(); err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			if err := db.CreatePanelQc(panelqc); err != nil {
				if mongo.IsDuplicateKeyError(err) {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Genepanel QC already exists for", "runid": panelqc.RunId})
					return
				}
				if errors.Is(err, mongo.ErrConflict) {
					c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": err.Error(), "run id": panelqc.RunId})
					return
				}
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "successfully added GenePanel QC data for", "id": panelqc.RunId})
		} else {
			c.JSON(http.StatusOK, gin.H{"message": "RunID contained no specific genepanel"})
		}
	}
}