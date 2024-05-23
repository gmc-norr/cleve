package gin

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type ChartConfig struct {
	ChartType string
	ChartData string
}

func (c ChartConfig) UrlParams() string {
	return fmt.Sprintf("?chart-data=%s&chart-type=%s", c.ChartData, c.ChartType)
}

func GetChartConfig(c *gin.Context) ChartConfig {
	return ChartConfig{
		ChartType: c.DefaultQuery("chart-type", "bar"),
		ChartData: c.DefaultQuery("chart-data", "q30"),
	}
}
