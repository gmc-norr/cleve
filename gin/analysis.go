package gin

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)

// Interface for reading analyses from the database.
type AnalysisGetter interface {
	Analyses(cleve.AnalysisFilter) (cleve.AnalysisResult, error)
	Analysis(analysisId string, parentId string) (*cleve.Analysis, error)
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

func AnalysesHandler(db AnalysisGetter, level cleve.AnalysisLevel) gin.HandlerFunc {
	var parentIdKey string
	switch level {
	case cleve.LevelRun:
		parentIdKey = "runId"
	case cleve.LevelCase:
		parentIdKey = "caseId"
	case cleve.LevelSample:
		parentIdKey = "sampleId"
	}
	return func(c *gin.Context) {
		filter, err := getAnalysisFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		filter.ParentId = c.Param(parentIdKey)
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

func AnalysisHandler(db AnalysisGetter, level cleve.AnalysisLevel) gin.HandlerFunc {
	var parentIdKey string
	switch level {
	case cleve.LevelRun:
		parentIdKey = "runId"
	case cleve.LevelCase:
		parentIdKey = "caseId"
	case cleve.LevelSample:
		parentIdKey = "sampleId"
	}
	return func(c *gin.Context) {
		parentId := c.Param(parentIdKey)
		analysisId := c.Param("analysisId")
		analysis, err := db.Analysis(analysisId, parentId)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.AbortWithStatusJSON(
					http.StatusNotFound,
					gin.H{
						"error":       "analysis not found",
						"parent_id":   parentId,
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
			AnalysisId      string               `json:"analysis_id" binding:"required"`
			ParentId        string               `json:"parent_id" binding:"required"`
			Level           cleve.AnalysisLevel  `json:"level" binding:"required"`
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
			ParentId:        params.ParentId,
			Level:           params.Level,
			Path:            params.Path,
			Software:        params.Software,
			SoftwareVersion: params.SoftwareVersion,
			Files:           params.Files,
		}
		a.StateHistory.Add(params.State)
		if a.Files == nil {
			a.Files = make([]cleve.AnalysisFile, 0)
		}

		// Check that the analysis doesn't already exist
		_, err := db.Analysis(a.AnalysisId, a.ParentId)
		if err == nil {
			c.AbortWithStatusJSON(
				http.StatusConflict,
				gin.H{
					"error":       "analysis already exists",
					"parent_id":   a.ParentId,
					"analysis_id": a.AnalysisId,
					"level":       a.Level,
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
			"parent_id":   a.ParentId,
			"analysis_id": a.AnalysisId,
			"level":       a.Level,
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

		slog.Debug("updating analysis", "payload", updateRequest)

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
