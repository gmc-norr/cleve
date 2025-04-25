package gin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve/charts"
	"github.com/gmc-norr/cleve/interop"
	"github.com/gmc-norr/cleve/mongo"
)

func GlobalChartsHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		config := GetChartConfig(c)
		filter, err := getQcFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		qc, err := db.RunQCs(filter)
		if errors.Is(err, mongo.ErrNoDocuments) || qc.Count == 0 {
			c.String(http.StatusOK, "No data to plot")
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		switch config.ChartData {
		case "q30":
			plotData := charts.RunStats[interop.OptionalFloat]{
				Data:  make([]charts.RunStat[interop.OptionalFloat], 0),
				Label: "%>=Q30",
				Type:  config.ChartType,
			}
			for _, q := range qc.InteropSummary {
				q30 := q.RunSummary.PercentQ30
				datapoint := charts.RunStat[interop.OptionalFloat]{
					RunID: q.RunId,
				}
				if !q30.IsNaN() {
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
			plotData := charts.RunStats[interop.OptionalFloat]{
				Data:  make([]charts.RunStat[interop.OptionalFloat], 0),
				Label: "Error rate",
				Type:  config.ChartType,
			}
			for _, q := range qc.InteropSummary {
				errorRate := q.RunSummary.ErrorRate
				datapoint := charts.RunStat[interop.OptionalFloat]{
					RunID: q.RunId,
				}
				if !errorRate.IsNaN() {
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
