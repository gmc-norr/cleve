package gin

import (
	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

func RunsHandler(c *gin.Context) {
	_, brief := c.GetQuery("brief")

	platform, _ := c.GetQuery("platform")
	state, _ := c.GetQuery("state")

	runs, err := mongo.GetRuns(brief, platform, state)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, runs)
}

func RunHandler(c *gin.Context) {
	runId := c.Param("runId")
	_, brief := c.GetQuery("brief")
	run, err := mongo.GetRun(runId, brief)

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

	var runParams cleve.RunParameters
	runParams, err = cleve.ParseRunParameters(paramsData)

	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid run parameters"})
		return
	}

	var state cleve.RunState
	if err = state.Set(addRunRequest.State); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	run := mongo.Run{
		RunID:          runParams.GetRunID(),
		ExperimentName: runParams.GetExperimentName(),
		Path:           addRunRequest.Path,
		Platform:       runParams.Platform(),
		RunParameters:  runParams,
		StateHistory:   []cleve.TimedRunState{{State: state, Time: time.Now()}},
		Analysis:       []*cleve.Analysis{},
	}

	if err := mongo.AddRun(&run); err != nil {
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

	var state cleve.RunState
	err := state.Set(updateRequest.State)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "when": "parsing state"})
		return
	}

	if err = mongo.UpdateRunState(runId, state); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "updating run state"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "run updated", "run_id": runId, "state": state.String()})
}
