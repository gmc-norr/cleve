package gin

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)

func AddSampleSheetHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		runID := c.Param("runId")

		var createRequest struct {
			SampleSheetPath string `json:"samplesheet" binding:"required"`
		}
		err := c.BindJSON(&createRequest)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		sampleSheet, err := cleve.ReadSampleSheet(createRequest.SampleSheetPath)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		sampleSheet.RunID = runID

		log.Printf("adding samplesheet for run %q", runID)

		res, err := db.SampleSheets.Create(runID, sampleSheet)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		fmt.Printf("%+v\n", res)

		switch {
		case res.MatchedCount == 0 && res.UpsertedCount == 1:
			c.JSON(http.StatusOK, gin.H{
				"message": fmt.Sprintf("added samplesheet for run %q", runID),
			})
		case res.MatchedCount == 1 && res.ModifiedCount == 1:
			c.JSON(http.StatusOK, gin.H{
				"message": fmt.Sprintf("updated samplesheet for run %q", runID),
			})
		case res.MatchedCount == 1 && res.ModifiedCount == 0:
			c.JSON(http.StatusBadRequest, gin.H{
				"message": fmt.Sprintf("samplesheet not updated; a newer samplesheet already exists for run %q", runID),
			})
		}
	}
}

func SampleSheetHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		runID := c.Param("runId")
		sampleSheet, err := db.SampleSheets.Get(runID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
					"error": fmt.Sprintf("no samplesheet found for run %q", runID),
				})
				return
			}
			log.Fatal(err)
		}

		c.JSON(http.StatusOK, sampleSheet)
	}
}
