package gin

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)

type RunQCGetter interface {
	Runs(cleve.RunFilter) (cleve.RunResult, error)
	RunQC(string) (*cleve.InteropQC, error)
	RunQCs(cleve.QcFilter) (cleve.QcResult, error)
}

type RunQCSetter interface {
	Run(string, bool) (*cleve.Run, error)
	CreateRunQC(string, *cleve.InteropQC) error
}

type RunQCGetterSetter interface {
	RunQCGetter
	RunQCSetter
}

func RunQcHandler(db RunQCGetter) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		runId := ctx.Param("runId")
		qc, err := db.RunQC(runId)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				ctx.AbortWithStatusJSON(
					http.StatusNotFound,
					gin.H{"error": fmt.Sprintf("qc for run %s not found", runId)})
				return
			}
			ctx.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": err.Error()},
			)
			return
		}
		ctx.JSON(http.StatusOK, qc)
	}
}

func AllRunQcHandler(db RunQCGetter) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		filter := cleve.RunFilter{
			Brief:    true,
			Platform: ctx.Param("platformName"),
			State:    cleve.Ready.String(),
		}
		runs, err := db.Runs(filter)
		if err != nil {
			ctx.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": err.Error()},
			)
		}

		runIds := make([]string, 0)
		for _, r := range runs.Runs {
			runIds = append(runIds, r.RunID)
		}

		qc := make([]*cleve.InteropQC, 0)

		for _, r := range runIds {
			qcSummary, err := db.RunQC(r)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					continue
				}
				log.Printf("warning: qc for run %s could not be fetched: %s", r, err.Error())
			}
			qc = append(qc, qcSummary)
		}
		ctx.JSON(http.StatusOK, qc)
	}
}

func AddRunQcHandler(db RunQCGetterSetter) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		runId := ctx.Param("runId")
		run, err := db.Run(runId, true)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				ctx.AbortWithStatusJSON(
					http.StatusNotFound,
					gin.H{"error": fmt.Sprintf("run %s not found", runId)},
				)
				return
			}
			ctx.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": fmt.Sprintf("run %s not found", runId)},
			)
			return
		}

		if len(run.StateHistory) == 0 || run.StateHistory[0].State != cleve.Ready {
			ctx.AbortWithStatusJSON(
				http.StatusNotFound,
				gin.H{"error": fmt.Sprintf("run %s not ready", runId)},
			)
			return
		}

		if _, err := db.RunQC(runId); err != mongo.ErrNoDocuments {
			ctx.AbortWithStatusJSON(
				http.StatusConflict,
				gin.H{"error": fmt.Sprintf("qc data already exists for run %s", runId)},
			)
			return
		}

		interopPath := fmt.Sprintf("%s/InterOp", run.Path)
		if _, err := os.Stat(interopPath); os.IsNotExist(err) {
			ctx.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": fmt.Sprintf("interop directory not found for run %s: %s", runId, interopPath)},
			)
			return
		}

		summary, err := cleve.GenerateSummary(run.Path)
		if err != nil {
			ctx.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": err.Error()},
			)
			return
		}

		imaging, err := cleve.GenerateImagingTable(runId, run.Path)
		if err != nil {
			ctx.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": err.Error()},
			)
			return
		}

		qc := &cleve.InteropQC{
			RunID:          runId,
			InteropSummary: summary,
			TileSummary:    imaging.LaneTileSummary(),
		}

		if err := db.CreateRunQC(runId, qc); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				ctx.AbortWithStatusJSON(
					http.StatusConflict,
					gin.H{"error": fmt.Sprintf("qc data already exists for run %s", runId)},
				)
				return
			}
			ctx.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": err.Error()},
			)
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("run qc data added for run %s", runId)})
	}
}
