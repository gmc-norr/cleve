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
	Analysis(string, string) (*cleve.Analysis, error)
}

// Interface for storing/updating analyses in the database.
type AnalysisSetter interface {
	CreateAnalysis(string, *cleve.Analysis) error
	SetAnalysisState(string, string, cleve.State) error
	SetAnalysisPath(string, string, string) error
}

// Interface for both getting and storing/updating analyses.
type AnalysisGetterSetter interface {
	AnalysisGetter
	AnalysisSetter
}

func RunAnalysesHandler(db AnalysisGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getAnalysisFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		filter.ParentId = c.Param("runId")
		slog.Debug("analysis filter", "filter", filter)
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

func RunAnalysisHandler(db AnalysisGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		runId := c.Param("runId")
		analysisId := c.Param("analysisId")
		analysis, err := db.Analysis(analysisId, runId)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.AbortWithStatusJSON(
					http.StatusNotFound,
					gin.H{
						"error":       "analysis not found",
						"run_id":      runId,
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
		runId := c.Param("runId")
		var addAnalysisRequest struct {
			Path        string                `form:"path" binding:"required"`
			State       string                `form:"state" binding:"required"`
			SummaryFile *multipart.FileHeader `form:"summary_file"`
		}

		if err := c.ShouldBind(&addAnalysisRequest); err != nil {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				gin.H{"error": err.Error(), "when": "parsing request body"},
			)
			return
		}

		var state cleve.State
		if err := state.Set(addAnalysisRequest.State); err != nil {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				gin.H{"error": err.Error(), "when": "parsing state"},
			)
			return
		}

		var summary *cleve.AnalysisSummary
		if addAnalysisRequest.SummaryFile != nil {
			summaryFile, err := addAnalysisRequest.SummaryFile.Open()
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

			s, err := cleve.ParseAnalysisSummary(summaryData)
			if err != nil {
				c.AbortWithStatusJSON(
					http.StatusBadRequest,
					gin.H{"error": err.Error(), "when": "parsing summary file"},
				)
				return
			}
			summary = &s
		}

		a := cleve.Analysis{
			AnalysisId: filepath.Base(addAnalysisRequest.Path),
			Path:       addAnalysisRequest.Path,
			State:      state,
			Summary:    summary,
		}

		// Check that the analysis doesn't already exist
		_, err := db.Analysis(runId, a.AnalysisId)
		if err == nil {
			c.AbortWithStatusJSON(
				http.StatusConflict,
				gin.H{
					"error":       "analysis already exists",
					"run_id":      runId,
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

		// Make sure that analyses are being added to the right run, and
		// also that requeued analyses are correctly identified. Only
		// check this if the analysis summary is actually present, otherwise
		// just add it blindly.
		requeRegex := regexp.MustCompile(`-Requeued-\d+$`)
		if a.Summary != nil && requeRegex.ReplaceAllString(a.Summary.RunID, "") != runId {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				gin.H{
					"error":           "run id in summary does not match the id of the run it is being added to",
					"run_id":          runId,
					"analysis_run_id": a.Summary.RunID,
				},
			)
			return
		}

		if err := db.CreateAnalysis(runId, &a); err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": err.Error(), "when": "adding analysis"},
			)
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":     "analysis added",
			"run_id":      runId,
			"analysis_id": a.AnalysisId,
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
