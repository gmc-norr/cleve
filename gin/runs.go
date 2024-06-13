package gin

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)

func RunsHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, brief := c.GetQuery("brief")
		filter, err := getRunFilter(c, brief)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		runs, err := db.Runs.All(filter)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, runs)
	}
}

func RunHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		runId := c.Param("runId")
		_, brief := c.GetQuery("brief")
		run, err := db.Runs.Get(runId, brief)

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
}

func AddRunHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var addRunRequest struct {
			Path  string `json:"path" binding:"required"`
			State string `json:"state" binding:"required"`
		}

		if err := c.BindJSON(&addRunRequest); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error(), "when": "parsing request body"})
			return
		}

		paramsFilename := filepath.Join(addRunRequest.Path, "RunParameters.xml")
		paramsFile, err := os.Open(paramsFilename)
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

		infoFilename := filepath.Join(addRunRequest.Path, "RunInfo.xml")
		infoFile, err := os.Open(infoFilename)
		defer infoFile.Close()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "opening run info file"})
			return
		}

		infoData, err := io.ReadAll(infoFile)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "reading run info file"})
			return
		}

		var runInfo cleve.RunInfo
		runInfo, err = cleve.ParseRunInfo(infoData)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid run info"})
			return
		}

		var state cleve.RunState
		if err = state.Set(addRunRequest.State); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		run := cleve.Run{
			RunID:          runParams.GetRunID(),
			ExperimentName: runParams.GetExperimentName(),
			Path:           addRunRequest.Path,
			Platform:       runParams.Platform(),
			RunParameters:  runParams,
			RunInfo:        runInfo,
			StateHistory:   []cleve.TimedRunState{{State: state, Time: time.Now()}},
			Analysis:       []*cleve.Analysis{},
		}

		// Check for a sspath
		sspath, err := cleve.MostRecentSamplesheet(run.Path)
		if err != nil {
			if err.Error() != "no samplesheet found" {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "looking for samplesheet"})
				return
			}
		}
		if sspath != "" {
			samplesheet, err := cleve.ReadSampleSheet(sspath)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "reading samplesheet"})
				return
			}
			_, err = db.SampleSheets.Create(run.RunID, samplesheet)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "saving samplesheet"})
				return
			}
		}

		// Save the run
		if err := db.Runs.Create(&run); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "run added", "run_id": run.RunID})
	}
}

func UpdateRunHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		if err = db.Runs.SetState(runId, state); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "updating run state"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "run updated", "run_id": runId, "state": state.String()})
	}
}
