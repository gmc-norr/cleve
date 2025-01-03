package gin

import (
	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
)

// TODO: find a better way of setting the defaults for the various filters.
// Possibly use a constructor for them.

func getRunFilter(c *gin.Context) (cleve.RunFilter, error) {
	// Set filter defaults
	filter := cleve.RunFilter{}
	filter.PageSize = 10
	if err := c.BindQuery(&filter); err != nil {
		return filter, err
	}
	return filter, filter.Validate()
}

func getSampleFilter(c *gin.Context) (cleve.SampleFilter, error) {
	// Set filter defaults
	filter := cleve.SampleFilter{}
	filter.PageSize = 10
	if err := c.BindQuery(&filter); err != nil {
		return filter, err
	}
	return filter, filter.Validate()
}

func getQcFilter(c *gin.Context) (cleve.QcFilter, error) {
	// Set filter defaults
	filter := cleve.QcFilter{}
	filter.PageSize = 5
	if err := c.BindQuery(&filter); err != nil {
		return filter, err
	}
	return filter, filter.Validate()
}
