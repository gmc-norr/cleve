package gin

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/interop"
	"github.com/gmc-norr/cleve/mongo"
)

func getDashboardData(db *mongo.DB, filter cleve.RunFilter) (gin.H, error) {
	runs, err := db.Runs.All(filter)
	if err != nil {
		return gin.H{"error": err.Error()}, err
	}

	platforms, err := db.Platforms.All()
	if err != nil {
		return gin.H{"error": err.Error()}, err
	}
	platformStrings := make([]string, 0)
	for _, p := range platforms {
		platformStrings = append(platformStrings, p.Name)
	}

	return gin.H{"runs": runs, "platforms": platformStrings, "run_filter": filter}, nil
}

func getRunFilter(c *gin.Context, brief bool) (cleve.RunFilter, error) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		return cleve.RunFilter{}, err
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil {
		return cleve.RunFilter{}, err
	}

	filter := cleve.RunFilter{
		Brief:    brief,
		RunID:    c.Query("run_id"),
		Platform: c.Query("platform"),
		State:    c.Query("state"),
		Page:     page,
		PageSize: pageSize,
	}

	return filter, nil
}

func DashboardHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getRunFilter(c, false)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		dashboardData, err := getDashboardData(db, filter)

		if err != nil {
			c.HTML(http.StatusInternalServerError, "error500", dashboardData)
		}

		c.Header("Hx-Push-Url", filter.UrlParams())
		c.HTML(http.StatusOK, "dashboard", dashboardData)
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

func DashboardRunTable(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getRunFilter(c, false)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		dashboardData, err := getDashboardData(db, filter)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error500", dashboardData)
		}

		c.Header("Hx-Push-Url", filter.UrlParams())
		c.HTML(http.StatusOK, "run_table", dashboardData)
	}
}

func DashboardQCHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getRunFilter(c, true)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		dashboardData, err := getDashboardData(db, filter)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error500", dashboardData)
			return
		}

		type RunDetails struct {
			Run *cleve.Run
			QC  *interop.InteropSummary
		}

		runDetails := make(map[string]RunDetails)

		for _, r := range dashboardData["runs"].(cleve.RunResult).Runs {
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

		c.HTML(http.StatusOK, "qc", gin.H{"run_details": runDetails, "platforms": dashboardData["platforms"]})
	}
}
