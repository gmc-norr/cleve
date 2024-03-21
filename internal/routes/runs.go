package routes

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
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
	runs, err := db.GetRuns()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, runs)
}

func RunHandler(c *gin.Context) {
	runId := c.Param("runId")
	run, err := db.GetRun(runId)

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
		RunParameterFile *multipart.FileHeader `form:"runparameters" binding:"required"`
	}

	if err := c.Bind(&addRunRequest); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	paramsFile, err := addRunRequest.RunParameterFile.Open()
	defer paramsFile.Close()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	paramsData, err := io.ReadAll(paramsFile)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var runParams runparameters.RunParameters
	runParams, err = runparameters.ParseRunParameters(paramsData)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid run parameters"})
		return
	}

	var state runstate.RunState
	if err = state.Set("new"); err != nil {
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
	}

	if err := db.AddRun(&run); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "run added", "run_id": run.RunID})
}

func UpdateRunHandler(c *gin.Context) {
	runId := c.Param("runId")

	var stateRequest struct {
		State *string `json:"state"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&stateRequest); err != nil {
		if err == io.EOF {
			c.JSON(http.StatusBadRequest, gin.H{"error": "empty request body"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if stateRequest.State == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state must be provided"})
		return
	}

	var state runstate.RunState
	err := state.Set(*stateRequest.State)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err = db.UpdateRunState(runId, state); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}
