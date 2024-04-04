package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve/analysis"
	"github.com/gmc-norr/cleve/internal/db"
	"github.com/gmc-norr/cleve/internal/db/runstate"
	"go.mongodb.org/mongo-driver/mongo"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
)

func AnalysesHandler(c *gin.Context) {
	runId := c.Param("runId")
	analyses, err := db.GetAnalyses(runId)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, analyses)
}

func AnalysisHandler(c *gin.Context) {
	runId := c.Param("runId")
	analysisId := c.Param("analysisId")
	analysis, err := db.GetAnalysis(runId, analysisId)
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

func AddAnalysisHandler(c *gin.Context) {
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

	var state runstate.RunState
	if err := state.Set(addAnalysisRequest.State); err != nil {
		c.AbortWithStatusJSON(
			http.StatusBadRequest,
			gin.H{"error": err.Error(), "when": "parsing state"},
		)
		return
	}

	var summary *analysis.AnalysisSummary
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

		s, err := analysis.ParseAnalysisSummary(summaryData)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusBadRequest,
				gin.H{"error": err.Error(), "when": "parsing summary file"},
			)
			return
		}
		summary = &s
	}

	a := analysis.Analysis{
		AnalysisId: filepath.Base(addAnalysisRequest.Path),
		Path:       addAnalysisRequest.Path,
		State:      state,
		Summary:    summary,
	}

	// Check that the analysis doesn't already exist
	_, err := db.GetAnalysis(runId, a.AnalysisId)
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

	if err := db.AddAnalysis(runId, &a); err != nil {
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

func UpdateAnalysisHandler(c *gin.Context) {
	runId := c.Param("runId")
	analysisId := c.Param("analysisId")

	var updateRequest struct {
		State       string                `form:"state"`
		SummaryFile *multipart.FileHeader `form:"summary_file"`
	}

	if err := c.Bind(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if updateRequest.State != "" {
		var state runstate.RunState
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

		err = db.UpdateAnalysisState(runId, analysisId, state)
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
		summary, err := analysis.ParseAnalysisSummary(summaryData)
		err = db.UpdateAnalysisSummary(runId, analysisId, &summary)
		if err != nil {
			c.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": err.Error(), "when": "updating analysis summary"},
			)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "analysis updated", "run_id": runId, "analysis_id": analysisId})
}
