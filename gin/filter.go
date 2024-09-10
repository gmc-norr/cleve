package gin

import (
	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
)

func getRunFilter(c *gin.Context) (cleve.RunFilter, error) {
	var filter cleve.RunFilter
	err := c.BindQuery(&filter)
	return filter, err
}

func getSampleFilter(c *gin.Context) (cleve.SampleFilter, error) {
	var filter cleve.SampleFilter
	err := c.BindQuery(&filter)
	return filter, err
}

func getQcFilter(c *gin.Context) (cleve.QcFilter, error) {
	var filter cleve.QcFilter
	err := c.BindQuery(&filter)
	return filter, err
}
