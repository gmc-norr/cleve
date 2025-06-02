package gin

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve/charts"
	"github.com/gmc-norr/cleve/mongo"
)

func IndexChartHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.Request.ParseForm(); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		runId := c.Param("runId")
		yData := c.Query("y")
		d, err := db.RunQC(runId)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		plotData := charts.RunStats[float64]{
			XLabel: "Sample",
			Type:   "bar",
		}
		switch yData {
		case "", "percent-pf-reads":
			plotData.YLabel = "%PF Reads"
		case "m-reads":
			plotData.YLabel = "Reads (M)"
		default:
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid data for y-axis: %s", yData)})
			return
		}
		for _, s := range d.IndexSummary.Indexes {
			datum := charts.RunStat[float64]{
				RunID: s.Sample,
			}
			switch yData {
			case "", "percent-pf-reads":
				datum.Value = &s.PercentReads
			case "m-reads":
				mReads := float64(s.ReadCount) / 1e6
				datum.Value = &mReads
			}
			plotData.Data = append(plotData.Data, datum)
		}

		p, err := plotData.Plot()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		s := p.RenderSnippet()
		c.String(http.StatusOK, s.Element+s.Script)
	}
}

func RunChartsHandler(db RunQCGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		runId := c.Param("runId")
		config := GetRunChartConfig(c)

		qc, err := db.RunQC(runId)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		plotData := charts.ScatterData[int]{
			XLimit: [2]float64{0, 100},
			YLimit: [2]float64{0, 100},
		}

		switch config.XData {
		case "percent_occupied":
			plotData.XLabel = "% occupied"
		case "percent_pf":
			plotData.XLabel = "% passing filter"
		}

		switch config.YData {
		case "percent_occupied":
			plotData.YLabel = "% occupied"
		case "percent_pf":
			plotData.YLabel = "% passing filter"
		}

		switch config.ColorBy {
		case "lane":
			plotData.Grouping = "Lane"
		}

		for _, tile := range qc.TileSummary {
			d := charts.ScatterDatum[int]{}
			switch config.XData {
			case "percent_occupied":
				d.X = float64(tile.PercentOccupied)
			case "percent_pf":
				d.X = float64(tile.PercentPF)
			}
			switch config.YData {
			case "percent_occupied":
				d.Y = float64(tile.PercentOccupied)
			case "percent_pf":
				d.Y = float64(tile.PercentPF)
			}

			switch config.ColorBy {
			case "lane":
				d.Group = tile.Lane
			}

			plotData.Data = append(plotData.Data, d)
		}

		switch config.ChartType {
		case "scatter":
			p, err := plotData.Plot()
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			s := p.RenderSnippet()
			c.String(http.StatusOK, s.Element+s.Script)
		}
	}
}
