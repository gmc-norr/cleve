package gin

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/interop"
	"github.com/gmc-norr/cleve/mongo"
)

// Interface for reading run QC data from the database.
type RunQCGetter interface {
	Runs(cleve.RunFilter) (cleve.RunResult, error)
	RunQC(string) (interop.InteropSummary, error)
	RunQCs(cleve.QcFilter) (cleve.QcResult, error)
}

// Interface for storing run QC data in the database.
type RunQCSetter interface {
	Run(string) (*cleve.Run, error)
	CreateRunQC(string, interop.InteropSummary) error
}

// Interface for both getting and storing run QC data.
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

func RunSamplesQcHandler(db RunQCGetter) gin.HandlerFunc {
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
		sampleQcs := make([]cleve.SampleQc, len(qc.IndexSummary.Indexes))
		for i, sample := range qc.IndexSummary.Indexes {
			sampleQcs[i] = cleve.SampleQc{
				SampleId:  sample.Sample,
				ReadCount: sample.ReadCount,
			}
		}
		ctx.JSON(http.StatusOK, sampleQcs)
	}
}

func AllRunQcHandler(db RunQCGetter) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		filter, err := getQcFilter(ctx)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		qc, err := db.RunQCs(filter)
		var oobError mongo.PageOutOfBoundsError
		if errors.As(err, &oobError) {
			ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": oobError.Error()})
			return
		}
		if errors.Is(err, mongo.ErrNoDocuments) {
			ctx.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, qc)
	}
}

func AddRunQcHandler(db RunQCGetterSetter) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		runId := ctx.Param("runId")
		run, err := db.Run(runId)
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

		if run.StateHistory.LastState() != cleve.StateReady {
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

		qc, err := interop.InteropFromDir(run.Path)
		if err != nil {
			ctx.AbortWithStatusJSON(
				http.StatusInternalServerError,
				gin.H{"error": fmt.Sprintf("failed to read interop data for %s: %s", runId, err)},
			)
			return
		}

		if err := db.CreateRunQC(runId, qc.Summarise()); err != nil {
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
