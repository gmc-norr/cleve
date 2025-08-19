package gin

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"math"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/interop"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/spf13/viper"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func authMiddleware(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestKey := c.Request.Header.Get("Authorization")
		if requestKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "missing authorization header",
			})
			return
		}

		_, err := db.Key(requestKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "invalid API key",
			})
			return
		}

		c.Next()
	}
}

func hxMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		hx := c.GetHeader("HX-Request")
		if hx != "true" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not an HX request"})
		}
		c.Next()
	}
}

func versionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-cleve-version", cleve.GetVersion())
		c.Next()
	}
}

func add(x, y float64) float64 {
	return x + y
}

func addInt(x, y int) int {
	return x + y
}

func subtract(x, y float64) float64 {
	return x - y
}

func subtractInt(x, y int) int {
	return x - y
}

func maxInt(x, y int) int {
	return max(x, y)
}

func minInt(x, y int) int {
	return min(x, y)
}

func multiply(x, y float64) float64 {
	return x * y
}

func multiplyInt(x, y int) int {
	return x * y
}

func title(s string) string {
	return cases.Title(language.English).String(s)
}

func N(start, end int) chan int {
	stream := make(chan int)
	go func() {
		for i := start; i < end; i++ {
			stream <- i
		}
		close(stream)
	}()
	return stream
}

func toFloat(x any) float64 {
	switch v := x.(type) {
	case float64:
		return v
	case interop.OptionalFloat:
		return float64(v)
	case int:
		return float64(v)
	}
	return 0
}

func LoadHTMLFS(e *gin.Engine, fs fs.FS, patterns ...string) {
	t := template.Must(template.New("").Funcs(e.FuncMap).ParseFS(fs, patterns...))
	e.SetHTMLTemplate(t)
}

func NewRouter(db *mongo.DB, debug bool) http.Handler {
	gin.DisableConsoleColor()
	if viper.GetString("logfile") != "" {
		f, err := os.Create(viper.GetString("logfile"))
		if err != nil {
			log.Fatal(err)
		}
		gin.DefaultWriter = io.MultiWriter(f, os.Stdout)
	}

	if debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.SetFuncMap(template.FuncMap{
		"add":         add,
		"addInt":      addInt,
		"subtract":    subtract,
		"subtractInt": subtractInt,
		"maxInt":      maxInt,
		"minInt":      minInt,
		"multiply":    multiply,
		"multiplyInt": multiplyInt,
		"title":       title,
		"toFloat":     toFloat,
		"isNaN":       math.IsNaN,
		"N":           N,
	})
	templateFS, err := cleve.GetTemplateFS()
	if err != nil {
		log.Fatalf("failed to get template fs: %s", err.Error())
	}
	LoadHTMLFS(r, templateFS, "*.tmpl")

	assetFS, err := cleve.GetAssetFS()
	if err != nil {
		log.Fatalf("failed to get asset fs: %s", err.Error())
	}
	r.StaticFS("/static", http.FS(assetFS))

	r.Use(versionMiddleware())

	// Dashboard endpoints
	r.GET("/", DashboardHandler(db))
	r.GET("/runs", DashboardHandler(db))
	r.GET("/runs/:runId", DashboardRunHandler(db))
	r.GET("/panels", DashboardPanelHandler(db))
	r.GET("/panels/:panelId", DashboardPanelHandler(db))
	r.GET("/qc", DashboardQCHandler(db))
	r.GET("/qc/charts/global", GlobalChartsHandler(db))
	r.GET("/qc/charts/run/:runId", RunChartsHandler(db))
	r.GET("/qc/charts/run/:runId/index", IndexChartHandler(db))

	hxEndpoints := r.Group("/")
	hxEndpoints.Use(hxMiddleware())
	hxEndpoints.GET("/runtable", DashboardRunTable(db))
	hxEndpoints.GET("/panel-list", DashboardPanelListHandler(db))

	// API endpoints
	r.GET("/api", func(c *gin.Context) {
		ApiIndexHandler(c, r.Routes())
	})

	r.GET("/api/runs", RunsHandler(db))
	r.GET("/api/runs/:runId", RunHandler(db))
	r.GET("/api/runs/:runId/analyses", RunAnalysesHandler(db))
	r.GET("/api/runs/:runId/analyses/:analysisId", RunAnalysisHandler(db))
	r.GET("/api/runs/:runId/samplesheet", RunSampleSheetHandler(db))
	r.GET("/api/runs/:runId/qc", RunQcHandler(db))
	r.GET("/api/panels", PanelsHandler(db))
	r.GET("/api/panels/:panelId", PanelHandler(db))
	r.GET("/api/platforms", PlatformsHandler(db))
	r.GET("/api/platforms/:platformName", GetPlatformHandler(db))
	r.GET("/api/qc/:platformName", AllRunQcHandler(db))
	r.GET("/api/samples", SamplesHandler(db))
	r.GET("/api/samples/:sampleId", SampleHandler(db))
	r.GET("/api/samplesheets/:uuid", SampleSheetHandler(db))

	authEndpoints := r.Group("/")
	authEndpoints.Use(authMiddleware(db))
	authEndpoints.POST("/api/analyses", AddAnalysisHandler(db))
	authEndpoints.PATCH("/api/analyses/:parentId/:analysisId", UpdateAnalysisHandler(db))
	authEndpoints.POST("/api/panels", AddPanelHandler(db))
	authEndpoints.PATCH("/api/panels/:panelId/archive", ArchivePanelHandler(db))
	authEndpoints.POST("/api/runs", AddRunHandler(db))
	authEndpoints.PATCH("/api/runs/:runId", UpdateRunHandler(db))
	authEndpoints.PATCH("/api/runs/:runId/path", UpdateRunPathHandler(db))
	authEndpoints.PATCH("/api/runs/:runId/state", UpdateRunStateHandler(db))
	authEndpoints.POST("/api/runs/:runId/samplesheet", AddRunSampleSheetHandler(db))
	authEndpoints.POST("/api/runs/:runId/qc", AddRunQcHandler(db))
	authEndpoints.POST("/api/samples", AddSampleHandler(db))
	authEndpoints.POST("/api/samplesheets", AddSampleSheetHandler(db))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			c.AbortWithStatusJSON(
				http.StatusNotFound,
				gin.H{
					"error": fmt.Sprintf("no such api endpoint for method %s", c.Request.Method),
					"code":  http.StatusNotFound,
				},
			)
			return
		}
		c.HTML(http.StatusNotFound, "error404", nil)
	})

	return r
}

func ApiIndexHandler(c *gin.Context, routes []gin.RouteInfo) {
	apiDocs, err := ParseAPIDocs(cleve.GetAPIDoc())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	c.HTML(http.StatusOK, "api.tmpl", apiDocs)
}
