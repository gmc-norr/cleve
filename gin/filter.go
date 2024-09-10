package gin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
)

func getPaginationFilter(c *gin.Context) (cleve.PaginationFilter, error) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		return cleve.PaginationFilter{}, err
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil {
		return cleve.PaginationFilter{}, err
	}
	filter := cleve.PaginationFilter{
		Page:     page,
		PageSize: pageSize,
	}
	return filter, nil
}

func getRunFilter(c *gin.Context, brief bool) (cleve.RunFilter, error) {
	filter := cleve.RunFilter{}
	paginationFilter, err := getPaginationFilter(c)
	if err != nil {
		return filter, err
	}

	filter.PaginationFilter = paginationFilter
	filter.Brief = brief
	filter.RunID = c.Query("run_id")
	filter.Platform = c.Query("platform")
	filter.State = c.Query("state")

	return filter, nil
}

func getSampleFilter(c *gin.Context) (cleve.SampleFilter, error) {
	filter := cleve.SampleFilter{}
	paginationFilter, err := getPaginationFilter(c)
	if err != nil {
		return filter, err
	}

	filter.PaginationFilter = paginationFilter
	filter.Id = c.Query("sample_id")
	filter.Name = c.Query("sample_name")

	return filter, nil
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
