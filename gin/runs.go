package gin

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)

// Interface for reading runs from the database.
type RunGetter interface {
	Run(string, bool) (*cleve.Run, error)
	Runs(cleve.RunFilter) (cleve.RunResult, error)
}

// Interface for storing/updating runs in the database.
type RunSetter interface {
	CreateRun(*cleve.Run) error
	CreateSampleSheet(cleve.SampleSheet, ...mongo.SampleSheetOption) (*cleve.UpdateResult, error)
	SetRunState(string, cleve.RunState) error
	SetRunPath(string, string) error
}

func RunsHandler(db RunGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getRunFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		runs, err := db.Runs(filter)

		if errors.As(err, &mongo.PageOutOfBoundsError{}) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, runs)
	}
}

func RunHandler(db RunGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		runId := c.Param("runId")
		_, brief := c.GetQuery("brief")
		run, err := db.Run(runId, brief)
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

func AddRunHandler(db RunSetter) gin.HandlerFunc {
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
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "opening run parameters file"})
			return
		}
		defer paramsFile.Close()

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
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "opening run info file"})
			return
		}
		defer infoFile.Close()

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
			_, err = db.CreateSampleSheet(samplesheet, mongo.SampleSheetWithRunId(run.RunID))
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "saving samplesheet"})
				return
			}
		}

		// Save the run
		if err := db.CreateRun(&run); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "run added", "run_id": run.RunID})
	}
}

func UpdateRunStateHandler(db RunSetter) gin.HandlerFunc {
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

		if err = db.SetRunState(runId, state); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "updating run state"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "run state updated", "run_id": runId, "state": state.String()})
	}
}

func UpdateRunPathHandler(db RunSetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		runId := c.Param("runId")

		var updateRequest struct {
			Path string `json:"path" binding:"required"`
		}

		if err := c.ShouldBindJSON(&updateRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "when": "parsing request body"})
			return
		}

		if err := db.SetRunPath(runId, updateRequest.Path); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "updating run path"})
			return
		}

		samplesheetPath, err := cleve.MostRecentSamplesheet(updateRequest.Path)
		if err != nil {
			if err.Error() != "no samplesheet found" {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "looking for samplesheet"})
				return
			}
		}

		if samplesheetPath != "" {
			samplesheet, err := cleve.ReadSampleSheet(samplesheetPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "reading samplesheet"})
				return
			}
			_, err = db.CreateSampleSheet(samplesheet, mongo.SampleSheetWithRunId(runId))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "saving samplesheet"})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "run path updated", "run_id": runId, "path": updateRequest.Path})
	}
}
