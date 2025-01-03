package gin

import (
	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
)

func getRunFilter(c *gin.Context) (cleve.RunFilter, error) {
	filter := cleve.NewRunFilter()
	if err := c.BindQuery(&filter); err != nil {
		return filter, err
	}
	return filter, filter.Validate()
}

func getSampleFilter(c *gin.Context) (cleve.SampleFilter, error) {
	filter := cleve.NewSampleFilter()
	if err := c.BindQuery(&filter); err != nil {
		return filter, err
	}
	return filter, filter.Validate()
}

func getQcFilter(c *gin.Context) (cleve.QcFilter, error) {
	filter := cleve.NewQcFilter()
	if err := c.BindQuery(&filter); err != nil {
		return filter, err
	}
	return filter, filter.Validate()
}
