package gin

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve/charts"
	"github.com/gmc-norr/cleve/mongo"
)

func GlobalChartsHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		config := GetChartConfig(c)
		filter, err := getQcFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		// Get all results
		filter.PageSize = 0

		qc, err := db.RunQCs(filter)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		switch config.ChartData {
		case "q30":
			plotData := charts.RunStats[float64]{
				Data:  make([]charts.RunStat[float64], 0),
				Label: "%>=Q30",
				Type:  config.ChartType,
			}
			for _, q := range qc.InteropSummary {
				q30 := q.RunSummary.PercentQ30
				datapoint := charts.RunStat[float64]{
					RunID: q.RunId,
				}
				if !math.IsNaN(q30) {
					datapoint.Value = &q30
				}
				plotData.Data = append(plotData.Data, datapoint)
			}
			p, err := plotData.Plot()
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if err := p.Render(c.Writer); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		case "error_rate":
			plotData := charts.RunStats[float64]{
				Data:  make([]charts.RunStat[float64], 0),
				Label: "Error rate",
				Type:  config.ChartType,
			}
			for _, q := range qc.InteropSummary {
				errorRate := q.RunSummary.ErrorRate
				datapoint := charts.RunStat[float64]{
					RunID: q.RunId,
				}
				if !math.IsNaN(errorRate) {
					datapoint.Value = &errorRate
				}
				plotData.Data = append(plotData.Data, datapoint)
			}
			p, err := plotData.Plot()
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if err := p.Render(c.Writer); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}
}
