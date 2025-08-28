package gin

import (
	"errors"
	"fmt"
	"log/slog"
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
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return gin.H{"error": err.Error()}, err
	}

	platforms, err := db.Platforms()
	if err != nil {
		return gin.H{"error": err.Error()}, err
	}
	platformNames := platforms.Names()

	return gin.H{"runs": runs.Runs, "metadata": runs.PaginationMetadata, "platforms": platformNames, "filter": filter, "cleve_version": cleve.GetVersion()}, nil
}

func DashboardHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getRunFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		dashboardData, err := getDashboardData(db, filter)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error500", gin.H{"error": err})
			return
		}

		var oobError mongo.PageOutOfBoundsError
		if errors.As(err, &oobError) {
			c.HTML(http.StatusNotFound, "error404", gin.H{"error": oobError})
			return
		}

		c.Header("Hx-Push-Url", filter.UrlParams())
		c.HTML(http.StatusOK, "dashboard", dashboardData)
	}
}

func DashboardPanelListHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.Request.ParseForm(); err != nil {
			slog.Error("failed to parse form data", "error", err)
		}
		filter, err := getPanelFilter(c)
		slog.Info("getting panel list", "filter", filter)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error500", gin.H{"error": err})
			c.Abort()
			return
		}
		categories, err := db.PanelCategories()
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error500", gin.H{"error": err})
			c.Abort()
			return
		}
		panels, err := db.Panels(filter)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error500", gin.H{"error": err})
			c.Abort()
			return
		}
		c.HTML(http.StatusOK, "panel-list", gin.H{"filter": filter, "categories": categories, "panels": panels})
	}
}

func DashboardPanelHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		panelId := c.Param("panelId")
		if err := c.Request.ParseForm(); err != nil {
			slog.Error("failed to parse form data", "error", err)
		}
		filter, err := getPanelFilter(c)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error500", gin.H{"error": err})
			c.Abort()
			return
		}
		d := gin.H{
			"cleve_version": cleve.GetVersion(),
		}
		panels, err := db.Panels(filter)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error500", gin.H{"error": err})
			c.Abort()
			return
		}
		categories, err := db.PanelCategories()
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error500", gin.H{"error": err})
			c.Abort()
			return
		}
		d["panels"] = panels
		d["categories"] = categories

		if panelId != "" {
			versions, err := db.PanelVersions(panelId)
			if err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					c.HTML(http.StatusNotFound, "error404", gin.H{"error": fmt.Sprintf("No panel found with id %q", panelId)})
					c.Abort()
					return
				}
				c.HTML(http.StatusInternalServerError, "error505", gin.H{"error": err})
				c.Abort()
				return
			}
			d["versions"] = versions
			if filter.Version == "" {
				filter.Version = versions[0].Version.String()
				d["version"] = filter.Version
			}

			panel, err := db.Panel(panelId, filter.Version)
			if err != nil {
				d["error"] = fmt.Sprintf("No panel with id %q was found", panelId)
				c.HTML(http.StatusNotFound, "error404", d)
				c.Abort()
				return
			}
			d["panel"] = panel
		}
		c.Header("HX-Push-Url", "/panels/"+panelId+"?version="+filter.Version)
		if c.GetHeader("HX-Request") == "true" {
			c.HTML(http.StatusOK, "panel-info", d)
			return
		}
		c.HTML(http.StatusOK, "panels", d)
	}
}

func DashboardRunHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		runId := c.Param("runId")
		run, err := db.Run(runId)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				c.HTML(http.StatusNotFound, "error404", gin.H{"error": fmt.Sprintf("run with id %q not found", runId)})
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

		c.HTML(http.StatusOK, "run", gin.H{"run": run, "qc": qc, "hasQc": hasQc, "samplesheet": sampleSheet, "chart_config": GetRunChartConfig(c), "cleve_version": cleve.GetVersion(), "message": message})
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
		if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
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
		if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
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
		c.HTML(http.StatusOK, "qc", gin.H{"qc": qc.InteropSummary, "metadata": qc.PaginationMetadata, "platforms": platformNames, "filter": filter, "chart_config": chartConfig, "cleve_version": cleve.GetVersion()})
	}
}
