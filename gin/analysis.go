package gin

import (
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"

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
	SetAnalysisState(string, string, cleve.State) error
	SetAnalysisPath(string, string, string) error
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

func UpdateAnalysisHandler(db AnalysisSetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		runId := c.Param("runId")
		analysisId := c.Param("analysisId")
		stateUpdated := false
		pathUpdated := false
		summaryUpdated := false

		var updateRequest struct {
			State       string                `form:"state"`
			Path        string                `form:"path"`
			SummaryFile *multipart.FileHeader `form:"summary_file"`
		}

		if err := c.Bind(&updateRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if updateRequest.State != "" {
			var state cleve.State
			err := state.Set(updateRequest.State)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					c.AbortWithStatusJSON(
						http.StatusBadRequest,
						gin.H{
							"error":       "analysis not found",
							"run_id":      runId,
							"analysis_id": analysisId,
						},
					)
					return
				}
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			err = db.SetAnalysisState(runId, analysisId, state)
			if err != nil {
				c.AbortWithStatusJSON(
					http.StatusInternalServerError,
					gin.H{
						"error": err.Error(),
						"when":  "updating analysis state",
					},
				)
				return
			}
			stateUpdated = true
		}

		if updateRequest.Path != "" {
			err := db.SetAnalysisPath(runId, analysisId, updateRequest.Path)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to set analysis path", "details": err})
			}
			pathUpdated = true
		}

		if updateRequest.SummaryFile != nil {
			summaryFile, err := updateRequest.SummaryFile.Open()
			if err != nil {
				c.AbortWithStatusJSON(
					http.StatusInternalServerError,
					gin.H{"error": err.Error(), "when": "opening summary file"},
				)
				return
			}
			summaryData, err := io.ReadAll(summaryFile)
			if err != nil {
				c.AbortWithStatusJSON(
					http.StatusInternalServerError,
					gin.H{"error": err.Error(), "when": "reading summary file"},
				)
				return
			}
			summary, err := cleve.ParseAnalysisSummary(summaryData)
			if err != nil {
				c.AbortWithStatusJSON(
					http.StatusInternalServerError,
					gin.H{"error": err.Error(), "when": "parsing analysis summary"},
				)
				return
			}
			err = db.SetAnalysisSummary(runId, analysisId, &summary)
			if err != nil {
				c.AbortWithStatusJSON(
					http.StatusInternalServerError,
					gin.H{"error": err.Error(), "when": "updating analysis summary"},
				)
				return
			}
			summaryUpdated = true
		}

		msg := "analysis updated"
		if !stateUpdated && !pathUpdated && !summaryUpdated {
			msg = "nothing updated"
		}

		c.JSON(http.StatusOK, gin.H{"message": msg, "run_id": runId, "analysis_id": analysisId, "updated_state": stateUpdated, "updated_path": pathUpdated, "updated_summary": summaryUpdated})
	}
}
