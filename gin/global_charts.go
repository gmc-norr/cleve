package gin

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/charts"
	"github.com/gmc-norr/cleve/mongo"
)

func GlobalChartsHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		platform := c.Query("platform")
		chartData := c.Query("chart-data")
		chartType := c.Query("chart-type")
		if chartType == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "chart type not set"})
		}
		if chartData == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "chart data not set"})
		}
		if platform == "all" {
			platform = ""
		}

		filter := cleve.RunFilter{
			Brief:    true,
			Platform: platform,
			State:    cleve.Ready.String(),
		}
		runs, err := db.Runs.All(filter)
		if err != nil {
			panic(err)
		}

		switch chartData {
		case "q30":
			plotData := charts.RunStats[float64]{
				Data:  make([]charts.RunStat[float64], 0),
				Label: "%>=Q30",
				Type:  chartType,
			}
			for _, r := range runs.Runs {
				q30, err := db.RunQC.GetTotalQ30(r.RunID)
				if err != nil {
					if err == mongo.ErrNoDocuments {
						continue
					}
				}

				datapoint := charts.RunStat[float64]{
					RunID: r.RunID,
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
			p.Render(c.Writer)
		case "error_rate":
			plotData := charts.RunStats[float64]{
				Data:  make([]charts.RunStat[float64], 0),
				Label: "Error rate",
				Type:  chartType,
			}
			for _, r := range runs.Runs {
				errorRate, err := db.RunQC.GetTotalErrorRate(r.RunID)
				if err != nil {
					if err == mongo.ErrNoDocuments {
						continue
					}
				}
				datapoint := charts.RunStat[float64]{
					RunID: r.RunID,
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
			p.Render(c.Writer)
		}
	}
}
