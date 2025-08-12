package gin

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/interop"
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
		if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
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
			Path string `json:"path" binding:"required"`
		}

		if err := c.BindJSON(&addRunRequest); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error(), "when": "parsing request body"})
			return
		}

		interopData, err := interop.InteropFromDir(addRunRequest.Path)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "when": "reading run information"})
			return
		}

		run := cleve.Run{
			RunID:          interopData.RunInfo.RunId,
			ExperimentName: interopData.RunParameters.ExperimentName,
			Path:           addRunRequest.Path,
			Platform:       interopData.RunInfo.Platform,
			RunParameters:  interopData.RunParameters,
			RunInfo:        interopData.RunInfo,
			Analysis:       []*cleve.Analysis{},
		}
		run.StateHistory.Add(run.State(false))

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

func UpdateRunHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		runId := c.Param("runId")
		var updateRequest struct {
			State          string `json:"state"`
			Path           string `json:"path"`
			UpdateMetadata bool   `json:"update_metadata"`
		}

		run, err := db.Run(runId, false)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.JSON(http.StatusNotFound, gin.H{"error": "run not found", "run_id": runId})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		updated := map[string]bool{
			"state":    false,
			"path":     false,
			"metadata": false,
		}

		if err := c.ShouldBindJSON(&updateRequest); err != nil {
			if !errors.Is(err, io.EOF) {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "when": "parsing request body"})
				return
			}
		}

		if updateRequest.Path != "" {
			if !filepath.IsAbs(updateRequest.Path) {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "run path must be absolute"})
				return
			}

			runinfo, err := interop.ReadRunInfo(filepath.Join(updateRequest.Path, "RunInfo.xml"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "when": "reading RunInfo.xml"})
				return
			}
			if runinfo.RunId != runId {
				c.JSON(http.StatusBadRequest, gin.H{"error": "run id at new path different from requested run", "when": "updating run path"})
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

			for i, a := range run.Analysis {
				if pathSuffix, found := strings.CutPrefix(a.Path, run.Path); found {
					run.Analysis[i].Path = filepath.Join(updateRequest.Path, pathSuffix)
				}
			}

			run.Path = updateRequest.Path
			updated["path"] = true
		}

		var state cleve.RunState
		if updateRequest.State != "" {
			err := state.Set(updateRequest.State)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "when": "parsing state"})
				return
			}
		} else {
			state = run.State(updated["path"])
		}

		if state != run.StateHistory.LastState().State {
			run.StateHistory.Add(state)
			updated["state"] = true
		}

		// Only update the metadata if the run has not been moved or is being moved
		if updateRequest.UpdateMetadata && !run.StateHistory.LastState().State.IsMoved() {
			runInfo, err := interop.ReadRunInfo(filepath.Join(run.Path, "RunInfo.xml"))
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "failed to read run info", "run": run.RunID, "error": err})
				return
			}
			runParameters, err := interop.ReadRunParameters(filepath.Join(run.Path, "RunParameters.xml"))
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "failed to read run parameters", "run": run.RunID, "error": err})
				return
			}
			run.RunInfo = runInfo
			run.RunParameters = runParameters
			updated["metadata"] = true
		}

		any_updated := false
		for _, u := range updated {
			if u {
				any_updated = true
				break
			}
		}

		if any_updated {
			if err := db.UpdateRun(run); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "failed to update run", "run": run.RunID, "error": err})
				return
			}
		}

		msg := "run updated"
		if !any_updated {
			msg = "nothing updated"
		}

		c.JSON(http.StatusOK, gin.H{"message": msg, "run_id": runId, "updated": updated})
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
