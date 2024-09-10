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
	filter := cleve.QcFilter{}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		return filter, err
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil {
		return filter, err
	}

	filter.RunID = c.Query("run_id")
	filter.Platform = c.Query("platform")
	filter.Page = page
	filter.PageSize = pageSize

	return filter, nil
}
