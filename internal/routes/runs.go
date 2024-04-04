package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve/analysis"
	"github.com/gmc-norr/cleve/internal/db"
	"github.com/gmc-norr/cleve/internal/db/runstate"
	"github.com/gmc-norr/cleve/runparameters"
	"go.mongodb.org/mongo-driver/mongo"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

func RunsHandler(c *gin.Context) {
	_, brief := c.GetQuery("brief")

	platform, _ := c.GetQuery("platform")
	state, _ := c.GetQuery("state")

	runs, err := db.GetRuns(brief, platform, state)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, runs)
}

func RunHandler(c *gin.Context) {
	runId := c.Param("runId")
	_, brief := c.GetQuery("brief")
	run, err := db.GetRun(runId, brief)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, run)
}

func AddRunHandler(c *gin.Context) {
	var addRunRequest struct {
		Path             string                `form:"path" binding:"required"`
		State            string                `form:"state" binding:"required"`
		RunParameterFile *multipart.FileHeader `form:"runparameters" binding:"required"`
	}

	if err := c.Bind(&addRunRequest); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error(), "when": "parsing request body"})
		return
	}

	paramsFile, err := addRunRequest.RunParameterFile.Open()
	defer paramsFile.Close()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "opening run parameters file"})
		return
	}

	paramsData, err := io.ReadAll(paramsFile)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "reading run parameters file"})
		return
	}

	var runParams runparameters.RunParameters
	runParams, err = runparameters.ParseRunParameters(paramsData)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid run parameters"})
		return
	}

	var state runstate.RunState
	if err = state.Set(addRunRequest.State); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	run := db.Run{
		RunID:          runParams.GetRunID(),
		ExperimentName: runParams.GetExperimentName(),
		Path:           addRunRequest.Path,
		Platform:       runParams.Platform(),
		RunParameters:  runParams,
		StateHistory:   []runstate.TimedRunState{{State: state, Time: time.Now()}},
		Analysis:       []*analysis.Analysis{},
	}

	if err := db.AddRun(&run); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "run added", "run_id": run.RunID})
}

func UpdateRunHandler(c *gin.Context) {
	runId := c.Param("runId")

	var updateRequest struct {
		State string `json:"state" binding:"required"`
	}

	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "when": "parsing request body"})
		return
	}

	var state runstate.RunState
	err := state.Set(updateRequest.State)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "when": "parsing state"})
		return
	}

	if err = db.UpdateRunState(runId, state); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "updating run state"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "run updated", "run_id": runId, "state": state.String()})
}
