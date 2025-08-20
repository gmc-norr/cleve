package gin

import (
	"errors"
	"io"

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

func getAnalysisFilter(c *gin.Context) (cleve.AnalysisFilter, error) {
	filter := cleve.NewAnalysisFilter()
	if err := c.BindQuery(&filter); err != nil && !errors.Is(err, io.EOF) {
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
	if p, ok := c.Params.Get("platformName"); ok {
		filter.Platform = p
	}
	return filter, filter.Validate()
}

func getPanelFilter(c *gin.Context) (cleve.PanelFilter, error) {
	filter := cleve.NewPanelFilter()
	if err := c.BindQuery(&filter); err != nil {
		return filter, err
	}
	return filter, nil
}
