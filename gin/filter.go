package gin

import (
	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
)

func getRunFilter(c *gin.Context) (cleve.RunFilter, error) {
	var filter cleve.RunFilter
	if err := c.BindQuery(&filter); err != nil {
		return filter, err
	}
	return filter, filter.Validate()
}

func getSampleFilter(c *gin.Context) (cleve.SampleFilter, error) {
	var filter cleve.SampleFilter
	if err := c.BindQuery(&filter); err != nil {
		return filter, err
	}
	return filter, filter.Validate()
}

func getQcFilter(c *gin.Context) (cleve.QcFilter, error) {
	var filter cleve.QcFilter
	if err := c.BindQuery(&filter); err != nil {
		return filter, err
	}
	return filter, filter.Validate()
}
