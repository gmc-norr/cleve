package gin

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)

// Interface for reading samples from the database.
type SampleGetter interface {
	Sample(string) (*cleve.Sample, error)
	Samples(filter *cleve.SampleFilter) (*cleve.SampleResult, error)
}

// Interface for storing/updating samples in the database.
type SampleSetter interface {
	CreateSample(*cleve.Sample) error
	CreateSamples([]*cleve.Sample) error
}

func SampleHandler(db SampleGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		sampleId := c.Param("sampleId")
		sample, err := db.Sample(sampleId)
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("sample with ID %s not found", sampleId)})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, sample)
	}
}

func SamplesHandler(db SampleGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter, err := getSampleFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		samples, err := db.Samples(&filter)
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusOK, samples)
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, samples)
	}
}

func AddSampleHandler(db SampleSetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		var addSampleRequest struct {
			Id       string                  `json:"id" binding:"required"`
			Name     string                  `json:"name" binding:"required"`
			Fastq    []string                `json:"fastq"`
			Analyses []*cleve.SampleAnalysis `json:"analyses"`
		}

		if err := c.BindJSON(&addSampleRequest); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		sample := cleve.Sample{
			Id:       addSampleRequest.Id,
			Name:     addSampleRequest.Name,
			Fastq:    addSampleRequest.Fastq,
			Analyses: addSampleRequest.Analyses,
		}

		if err := db.CreateSample(&sample); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "sample added", "sample_id": sample.Id})
	}
}
