package gin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)

type UserMessage struct {
	Text  string
	Class string
}

func NewUserMessage(text, class string) UserMessage {
	return UserMessage{
		Text:  text,
		Class: class,
	}
}

func getDashboardData(db *mongo.DB, filter cleve.RunFilter) (gin.H, error) {
	runs, err := db.Runs(filter)
	if err != nil {
		return gin.H{"error": err.Error()}, err
	}

	platforms, err := db.Platforms()
	if err != nil {
		return gin.H{"error": err.Error()}, err
	}
	platformNames := platforms.Names()

	return gin.H{"runs": runs, "platforms": platformNames, "run_filter": filter}, nil
}

func DashboardHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getRunFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		dashboardData, err := getDashboardData(db, filter)
		dashboardData["version"] = cleve.GetVersion()

		var oobError mongo.PageOutOfBoundsError
		if errors.As(err, &oobError) {
			c.HTML(http.StatusNotFound, "error404", gin.H{"error": oobError})
			return
		}
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error500", dashboardData)
			return
		}

		c.Header("Hx-Push-Url", filter.UrlParams())
		c.HTML(http.StatusOK, "dashboard", dashboardData)
	}
}

func DashboardRunHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		runId := c.Param("runId")
		run, err := db.Run(runId, false)
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
		qc, err := db.RunQC(runId)
		if err != nil {
			if err != mongo.ErrNoDocuments {
				c.HTML(http.StatusInternalServerError, "error500", gin.H{"error": err.Error()})
				c.Abort()
				return
			}
			hasQc = false
		}

		var message *UserMessage
		if hasQc && qc.Date.IsZero() {
			m := NewUserMessage(
				"The stored QC data for this run is of an unsupported version. Update this run to view it.",
				"warning",
			)
			message = &m
			hasQc = false
		}

		sampleSheet, err := db.SampleSheet(mongo.SampleSheetWithRunId(runId))
		if err != nil {
			if err != mongo.ErrNoDocuments {
				c.HTML(http.StatusInternalServerError, "error500", gin.H{"error": err.Error()})
				c.Abort()
				return
			}
		}

		c.HTML(http.StatusOK, "run", gin.H{"run": run, "qc": qc, "hasQc": hasQc, "samplesheet": sampleSheet, "chart_config": GetRunChartConfig(c), "version": cleve.GetVersion(), "message": message})
	}
}

func DashboardRunTable(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getRunFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		dashboardData, err := getDashboardData(db, filter)
		var oobError mongo.PageOutOfBoundsError
		if errors.As(err, &oobError) {
			c.HTML(http.StatusNotFound, "error404", dashboardData)
			return
		}
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error500", dashboardData)
			return
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

		qc, err := db.RunQCs(filter)
		var oobError mongo.PageOutOfBoundsError
		if errors.As(err, &oobError) {
			c.HTML(http.StatusNotFound, "error404", gin.H{"error": oobError.Error()})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		platforms, err := db.Platforms()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		platformNames := platforms.Names()

		c.Header("Hx-Push-Url", filter.UrlParams())
		c.HTML(http.StatusOK, "qc", gin.H{"qc": qc.InteropSummary, "metadata": qc.PaginationMetadata, "platforms": platformNames, "filter": filter, "chart_config": chartConfig, "version": cleve.GetVersion()})
	}
}
