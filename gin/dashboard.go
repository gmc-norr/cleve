package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
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

		sampleSheet, err := db.SampleSheets.Get(runId)
		if err != nil {
			if err != mongo.ErrNoDocuments {
				c.HTML(http.StatusInternalServerError, "error500", gin.H{"error": err.Error()})
				c.Abort()
				return
			}
		}

		c.HTML(http.StatusOK, "run", gin.H{"run": run, "qc": qc, "hasQc": hasQc, "samplesheet": sampleSheet})
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
		chartConfig := GetChartConfig(c)
		filter, err := getQcFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		qc, err := db.RunQC.All(filter)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		platforms, err := db.Platforms.All()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		platformStrings := make([]string, 0)
		for _, p := range platforms {
			platformStrings = append(platformStrings, p.Name)
		}

		c.Header("Hx-Push-Url", filter.UrlParams())
		c.HTML(http.StatusOK, "qc", gin.H{"qc": qc.Qc, "metadata": qc.RunMetadata, "platforms": platformStrings, "filter": filter, "chart-config": chartConfig})
	}
}
