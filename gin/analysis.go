package gin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)

// Interface for reading analyses from the database.
type AnalysisGetter interface {
	Analyses(cleve.AnalysisFilter) (cleve.AnalysisResult, error)
	Analysis(analysisId string) (*cleve.Analysis, error)
}

// Interface for storing/updating analyses in the database.
type AnalysisSetter interface {
	CreateAnalysis(*cleve.Analysis) error
	SetAnalysisState(analysisId string, parentId string, state cleve.State) error
	SetAnalysisPath(analysisId string, parentId string, path string) error
	SetAnalysisFiles(analysisId string, parentId string, files []cleve.AnalysisFile) error
}

// Interface for both getting and storing/updating analyses.
type AnalysisGetterSetter interface {
	AnalysisGetter
	AnalysisSetter
}

func AnalysesHandler(db AnalysisGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getAnalysisFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		analyses, err := db.Analyses(filter)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.JSON(http.StatusOK, analyses)
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, analyses)
	}
}

func AnalysisHandler(db AnalysisGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		analysisId := c.Param("analysisId")
		analysis, err := db.Analysis(analysisId)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.AbortWithStatusJSON(
					http.StatusNotFound,
					gin.H{
						"error":       "analysis not found",
						"analysis_id": analysisId,
					},
				)
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, analysis)
	}
}

func AddAnalysisHandler(db AnalysisGetterSetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params struct {
			Path            string               `json:"path" binding:"required"`
			RunId           string               `json:"run_id" binding:"required"`
			AnalysisId      string               `json:"analysis_id" binding:"required"`
			State           cleve.State          `json:"state" binding:"required"`
			Software        string               `json:"software" binding:"required"`
			SoftwareVersion string               `json:"software_version" binding:"required"`
			Files           []cleve.AnalysisFile `json:"files"`
		}

		if err := c.ShouldBind(&params); err != nil {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				gin.H{"error": err.Error(), "when": "parsing request body"},
			)
			return
		}

		a := cleve.Analysis{
			AnalysisId:      params.AnalysisId,
			Runs:            []string{params.RunId},
			Path:            params.Path,
			Software:        params.Software,
			SoftwareVersion: params.SoftwareVersion,
			OutputFiles:     params.Files,
		}
		a.StateHistory.Add(params.State)
		if a.OutputFiles == nil {
			a.OutputFiles = make([]cleve.AnalysisFile, 0)
		}

		// Check that the analysis doesn't already exist
		_, err := db.Analysis(a.AnalysisId)
		if err == nil {
			c.AbortWithStatusJSON(
				http.StatusConflict,
				gin.H{
					"error":       "analysis already exists",
					"analysis_id": a.AnalysisId,
				},
			)
			return
		} else if err != mongo.ErrNoDocuments {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{
					"error": err.Error(),
					"when":  "checking if analysis already exists",
				},
			)
			return
		}

		if err := db.CreateAnalysis(&a); err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": err.Error(), "when": "adding analysis"},
			)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":     "analysis added",
			"analysis_id": a.AnalysisId,
		})
	}
}

func UpdateAnalysisHandler(db AnalysisSetter, level cleve.AnalysisLevel) gin.HandlerFunc {
	var parentIdKey string
	switch level {
	case cleve.LevelRun:
		parentIdKey = "runId"
	case cleve.LevelCase:
		parentIdKey = "caseId"
	case cleve.LevelSample:
		parentIdKey = "sampleId"
	default:
		parentIdKey = "parentId"
	}
	return func(c *gin.Context) {
		parentId := c.Param(parentIdKey)
		analysisId := c.Param("analysisId")
		stateUpdated := false
		pathUpdated := false
		filesUpdated := false

		var updateRequest struct {
			State cleve.State          `json:"state"`
			Path  string               `json:"path"`
			Files []cleve.AnalysisFile `json:"files"`
		}

		if err := c.BindJSON(&updateRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if updateRequest.State.IsValid() {
			err := db.SetAnalysisState(analysisId, parentId, updateRequest.State)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
						"error":       "analysis not found",
						"parent_id":   parentId,
						"analysis_id": analysisId,
						"level":       level,
					})
					return
				}
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error":   "failed to set analysis state",
					"details": err,
				})
				return
			}
			stateUpdated = true
		}

		if updateRequest.Path != "" {
			err := db.SetAnalysisPath(analysisId, parentId, updateRequest.Path)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
						"error":       "analysis not found",
						"parent_id":   parentId,
						"analysis_id": analysisId,
						"level":       level,
					})
					return
				}
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":   "failed to set analysis path",
					"details": err,
				})
				return
			}
			pathUpdated = true
		}

		if len(updateRequest.Files) > 0 {
			err := db.SetAnalysisFiles(analysisId, parentId, updateRequest.Files)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
						"error":       "analysis not found",
						"parent_id":   parentId,
						"analysis_id": analysisId,
						"level":       level,
					})
					return
				}
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":   "failed to set analysis path",
					"details": err,
				})
				return
			}
			filesUpdated = true
		}

		msg := "analysis updated"
		if !stateUpdated && !pathUpdated && !filesUpdated {
			msg = "nothing updated"
		}

		c.JSON(http.StatusOK, gin.H{
			"message":       msg,
			"parent_id":     parentId,
			"analysis_id":   analysisId,
			"updated_state": stateUpdated,
			"updated_path":  pathUpdated,
			"updated_files": filesUpdated,
		})
	}
}
