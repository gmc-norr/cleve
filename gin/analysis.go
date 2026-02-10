package gin

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)

// Interface for reading analyses from the database.
type AnalysisGetter interface {
	Analyses(cleve.AnalysisFilter) (cleve.AnalysisResult, error)
	AnalysesFiles(cleve.AnalysisFileFilter) ([]cleve.AnalysisFile, error)
	Analysis(analysisId string, runId ...string) (*cleve.Analysis, error)
}

// Interface for storing/updating analyses in the database.
type AnalysisSetter interface {
	CreateAnalysis(*cleve.Analysis) error
	SetAnalysisState(analysisId string, state cleve.State) error
	SetAnalysisPath(analysisId string, path string) error
	SetAnalysisFiles(analysisId string, files []cleve.AnalysisFile) error
}

// Interface for both getting and storing/updating analyses.
type AnalysisGetterSetter interface {
	AnalysisGetter
	AnalysisSetter
}

func AnalysesHandler(db AnalysisGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		runId := c.Param("runId")
		filter, err := getAnalysisFilter(c)
		if runId != "" {
			filter.RunId = runId
		}
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
			if mongo.IsRegexError(err) {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, analyses)
	}
}

func AnalysesFileHandler(db AnalysisGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getAnalysisFileFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		files, err := db.AnalysesFiles(filter)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, files)
	}
}

func AnalysisHandler(db AnalysisGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		analysisId := c.Param("analysisId")
		runId := c.Param("runId")
		analysis, err := db.Analysis(analysisId, runId)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				payload := gin.H{
					"error":       "analysis not found",
					"analysis_id": analysisId,
				}
				if runId != "" {
					payload["run_id"] = runId
				}
				c.AbortWithStatusJSON(
					http.StatusNotFound,
					payload,
				)
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, analysis)
	}
}

func AnalysisFileHandler(db AnalysisGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getAnalysisFileFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		files, err := db.AnalysesFiles(filter)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, files)
	}
}

func AddAnalysisHandler(db AnalysisGetterSetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var params struct {
			Path            string                     `json:"path" binding:"required"`
			RunId           string                     `json:"run_id" binding:"required"`
			AnalysisId      string                     `json:"analysis_id" binding:"required"`
			State           cleve.State                `json:"state" binding:"required"`
			Software        string                     `json:"software" binding:"required"`
			SoftwareVersion string                     `json:"software_version" binding:"required"`
			InputFiles      []cleve.AnalysisFileFilter `json:"input_files"`
			OutputFiles     []cleve.AnalysisFile       `json:"output_files"`
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
			InputFiles:      params.InputFiles,
			OutputFiles:     params.OutputFiles,
		}
		a.StateHistory.Add(params.State)

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

		// Check the input files
		for _, f := range a.InputFiles {
			if err := f.Validate(); err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error":   "invalid input file entry",
					"details": err.Error(),
					"file":    f,
				})
				return
			} else if f.AnalysisId == "" {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error":   "analysis id cannot be empty",
					"details": "the analysis id must be defined for input files",
					"file":    f,
				})
				return
			}
		}

		// Check the output files
		for i := range a.OutputFiles {
			// Files should be part of the analysis, and thus the paths should be relative.
			a.OutputFiles[i].IsPartOfAnalysis()
			if err := a.OutputFiles[i].Validate(); err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error":   "invalid output file entry",
					"details": err.Error(),
					"file":    a.OutputFiles[i],
				})
				return
			}
		}

		if err := db.CreateAnalysis(&a); err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": err.Error(), "when": "adding analysis"},
			)
			return
		}

		c.Set("webhook_message", cleve.WebhookMessageRequest{
			Entity:      &a,
			Message:     "new analysis added",
			MessageType: cleve.MessageStateUpdate,
		})

		c.JSON(http.StatusOK, gin.H{
			"message":     "analysis added",
			"analysis_id": a.AnalysisId,
		})
	}
}

func UpdateAnalysisHandler(db AnalysisGetterSetter) gin.HandlerFunc {
	return func(c *gin.Context) {
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
			err := db.SetAnalysisState(analysisId, updateRequest.State)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
						"error":       "analysis not found",
						"analysis_id": analysisId,
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
			err := db.SetAnalysisPath(analysisId, updateRequest.Path)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
						"error":       "analysis not found",
						"analysis_id": analysisId,
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
			for i := range updateRequest.Files {
				// Files should be part of the analysis, and thus the paths should be relative.
				updateRequest.Files[i].IsPartOfAnalysis()
				if err := updateRequest.Files[i].Validate(); err != nil {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
						"error":   "invalid file entry",
						"details": err.Error(),
						"file":    updateRequest.Files[i],
					})
					return
				}
			}
			err := db.SetAnalysisFiles(analysisId, updateRequest.Files)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
						"error":       "analysis not found",
						"analysis_id": analysisId,
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

		if stateUpdated {
			a, err := db.Analysis(analysisId)
			if err != nil {
				_ = c.Error(fmt.Errorf("failed to fetch analysis when requesting web hook message"))
			} else {
				c.Set("webhook_message", cleve.WebhookMessageRequest{
					Entity:      a,
					Message:     "analysis state updated",
					MessageType: cleve.MessageStateUpdate,
				})
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message":       msg,
			"analysis_id":   analysisId,
			"updated_state": stateUpdated,
			"updated_path":  pathUpdated,
			"updated_files": filesUpdated,
		})
	}
}
