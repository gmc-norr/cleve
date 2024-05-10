package gin

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/interop"
	"github.com/gmc-norr/cleve/mongo"
)

func RunQcHandler(db *mongo.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		runId := ctx.Param("runId")
		qc, err := db.RunQC.Get(runId)
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

func AllQcHandler(db *mongo.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		platform := ctx.Param("platformName")
		runs, err := db.Runs.All(true, platform, cleve.Ready.String())
		if err != nil {
			ctx.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": err.Error()},
			)
		}

		runIds := make([]string, 0)
		for _, r := range runs {
			runIds = append(runIds, r.RunID)
		}

		qc := make([]*interop.InteropSummary, 0)

		for _, r := range runIds {
			qcSummary, err := db.RunQC.Get(r)
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

func AddRunQcHandler(db *mongo.DB) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		runId := ctx.Param("runId")
		run, err := db.Runs.Get(runId, true)
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

		if _, err := db.RunQC.Get(runId); err != mongo.ErrNoDocuments {
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
				gin.H{"error": fmt.Sprintf("interop directory not found for run %s", runId)},
			)
			return
		}

		qc, err := interop.GenerateSummary(runId, run.Path)
		if err != nil {
			ctx.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": err.Error()},
			)
			return
		}

		if err := db.RunQC.Create(runId, qc); err != nil {
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
