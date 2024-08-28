package gin

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve"
	"github.com/gmc-norr/cleve/mongo"
)

type SampleSheetGetter interface {
	SampleSheet(string) (cleve.SampleSheet, error)
}

type SampleSheetSetter interface {
	CreateSampleSheet(string, cleve.SampleSheet) (*cleve.UpdateResult, error)
}

func AddSampleSheetHandler(db SampleSheetSetter) gin.HandlerFunc {
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

		res, err := db.CreateSampleSheet(runID, sampleSheet)
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

func SampleSheetHandler(db SampleSheetGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		runID := c.Param("runId")
		sectionName := c.Query("section")
		columnName := c.QueryArray("column")
		key := c.Query("key")

		if key != "" && columnName != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "give key or column, not both",
			})
			return
		}

		if sectionName == "" && (key != "" || columnName != nil) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "section required when using key or column",
			})
			return
		}

		sampleSheet, err := db.SampleSheet(runID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
					"error": fmt.Sprintf("no samplesheet found for run %q", runID),
				})
				return
			}
			log.Fatal(err)
		}

		if sectionName != "" {
			section := sampleSheet.Section(sectionName)
			if section == nil {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
					"error": fmt.Sprintf("section %q not found in samplesheet", sectionName),
				})
				return
			}

			if columnName != nil {
				colData := make(map[string][]string)
				for _, colName := range columnName {
					col, err := section.GetColumn(colName)
					if err != nil {
						c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
							"error": err.Error(),
						})
						return
					}
					colData[colName] = col
				}
				c.JSON(http.StatusOK, colData)
				return
			}

			if key != "" {
				val, err := section.Get(key)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
						"error": err.Error(),
					})
					return
				}
				c.JSON(http.StatusOK, val)
				return
			}
			c.JSON(http.StatusOK, section)
			return
		}

		c.JSON(http.StatusOK, sampleSheet)
	}
}
