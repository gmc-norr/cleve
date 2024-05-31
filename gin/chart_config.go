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

type RunChartConfig struct {
	ChartType string
	XData     string
	YData     string
	ColorBy   string
}

func (c RunChartConfig) UrlParams() string {
	return fmt.Sprintf("?chart-type=%s&chart-data-x=%s&chart-data-y=%s", c.ChartType, c.XData, c.YData)
}

func GetRunChartConfig(c *gin.Context) RunChartConfig {
	return RunChartConfig{
		ChartType: c.DefaultQuery("chart-type", "scatter"),
		XData:     c.DefaultQuery("chart-data-x", "percent_occupancy"),
		YData:     c.DefaultQuery("chart-data-y", "percent_pf"),
		ColorBy:   c.DefaultQuery("chart-color-by", "lane"),
	}
}
