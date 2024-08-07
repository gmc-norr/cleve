package gin

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
)

func getRunFilter(c *gin.Context, brief bool) (cleve.RunFilter, error) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		return cleve.RunFilter{}, err
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if err != nil {
		return cleve.RunFilter{}, err
	}

	filter := cleve.RunFilter{
		Brief:    brief,
		RunID:    c.Query("run_id"),
		Platform: c.Query("platform"),
		State:    c.Query("state"),
		Page:     page,
		PageSize: pageSize,
	}

	return filter, nil
}
