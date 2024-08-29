package gin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve/charts"
	"github.com/gmc-norr/cleve/mongo"
)

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

		plotData := charts.ScatterData{}

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
			d := charts.ScatterDatum{}
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
				d.Color = tile.Lane
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
			p.Render(c.Writer)
		}
	}
}
