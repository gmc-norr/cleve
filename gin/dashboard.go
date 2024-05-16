package gin

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/interop"
	"github.com/gmc-norr/cleve/mongo"
)

func DashboardHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		runs, err := db.Runs.All(false, "", "")
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error500", gin.H{"error": err.Error()})
			return
		}
		c.HTML(http.StatusOK, "dashboard", gin.H{"runs": runs})
	}
}

func DashboardRunHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		runId := c.Param("runId")
		run, err := db.Runs.Get(runId, false)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.HTML(http.StatusNotFound, "error404", gin.H{"error": "run not found"})
				c.Abort()
				return
			}
			c.HTML(http.StatusInternalServerError, "error500", gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		hasQc := true
		qc, err := db.RunQC.Get(runId)
		if err != nil {
			if err != mongo.ErrNoDocuments {
				c.HTML(http.StatusInternalServerError, "error500", gin.H{"error": err.Error()})
				c.Abort()
				return
			}
			hasQc = false
		}

		c.HTML(http.StatusOK, "run", gin.H{"run": run, "qc": qc, "hasQc": hasQc})
	}
}

func DashboardQCHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		runs, err := db.Runs.All(true, "", "")
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error500", nil)
		}

		type RunDetails struct {
			Run *cleve.Run
			QC  *interop.InteropSummary
		}

		runDetails := make(map[string]RunDetails)

		for _, r := range runs {
			qcSummary, err := db.RunQC.Get(r.RunID)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					continue
				}
				log.Println(err)
				log.Printf("warning: qc for run %s could not be fetched: %s", r.RunID, err.Error())
			}
			runDetails[r.RunID] = RunDetails{
				Run: r,
				QC:  qcSummary,
			}
		}

		platforms, err := db.Platforms.All()
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error500", gin.H{"error": err.Error()})
		}
		platformStrings := make([]string, 0)
		for _, p := range platforms {
			platformStrings = append(platformStrings, p.Name)
		}

		c.HTML(http.StatusOK, "qc", gin.H{"run_details": runDetails, "platforms": platformStrings})
	}
}
