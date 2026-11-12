package gin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gmc-norr/cleve/charts"
	"github.com/gmc-norr/cleve/interop"
	"github.com/gmc-norr/cleve/mongo"
	"github.com/gmc-norr/cleve"
)

func GlobalChartsHandler(db *mongo.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		config := GetChartConfig(c)
		filter, err := getQcFilter(c)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		qc, err := db.RunQCs(filter)
		if errors.Is(err, mongo.ErrNoDocuments) || qc.Count == 0 {
			c.String(http.StatusOK, "No data to plot")
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		switch config.ChartData {
		case "q30":
			plotData := charts.RunStats[interop.OptionalFloat]{
				Data:  make([]charts.RunStat[interop.OptionalFloat], 0),
				Label: "%>=Q30",
				Type:  config.ChartType,
			}
			for _, q := range qc.InteropSummary {
				q30 := q.RunSummary.PercentQ30
				datapoint := charts.RunStat[interop.OptionalFloat]{
					RunID: q.RunId,
				}
				if !q30.IsNaN() {
					datapoint.Value = &q30
				}
				plotData.Data = append(plotData.Data, datapoint)
			}
			p, err := plotData.Plot()
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if err := p.Render(c.Writer); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		case "error_rate":
			plotData := charts.RunStats[interop.OptionalFloat]{
				Data:  make([]charts.RunStat[interop.OptionalFloat], 0),
				Label: "Error rate",
				Type:  config.ChartType,
			}
			for _, q := range qc.InteropSummary {
				errorRate := q.RunSummary.ErrorRate
				datapoint := charts.RunStat[interop.OptionalFloat]{
					RunID: q.RunId,
				}
				if !errorRate.IsNaN() {
					datapoint.Value = &errorRate
				}
				plotData.Data = append(plotData.Data, datapoint)
			}
			p, err := plotData.Plot()
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if err := p.Render(c.Writer); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}
}


func PanelGeneQcChartHandler(db PanelQCGetter) gin.HandlerFunc {
	return func(c *gin.Context) {

		genePanelId := c.Param("genepanalId")
		gene := c.Query("gene"+genePanelId)
		gene_qcdata := c.Query("gene_qcdata"+genePanelId)

		geneqc, err := db.PanelGeneQc(gene, genePanelId)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		metric, ok := geneMetrics[gene_qcdata]
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Cannot retrieve the metrice data"})
			return
		}

		p, err := charts.GeneQCLineChart(
			geneqc.Genes, 
			gene_qcdata, 
			func(g *cleve.GeneQcrunid) string { return g.RunId },
			metric,
		)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	
		s := p.RenderSnippet()
		c.String(http.StatusOK, s.Element+s.Script)

	}
}

func GeneQcChartHandler(db GeneQCGetter) gin.HandlerFunc {
	return func(c *gin.Context) {

		gene := c.Query("gene")
		gene_qcdata := c.Query("gene_qcdata")

		geneqc, err := db.GeneQc(gene)
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.String(http.StatusOK, "No data to plot")
			return
		}

		metric, ok := geneMetrics[gene_qcdata]
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Cannot retrieve the metrice data"})
			return
		}

		p, err := charts.GeneQCLineChart(
			geneqc.Genes, 
			gene_qcdata, 
			func(g *cleve.GeneQcrunid) string { return g.RunId },
			metric,
		)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	
		s := p.RenderSnippet()
		c.String(http.StatusOK, s.Element+s.Script)

	}
}

func ExonQcChartHandler(db GeneQCGetter) gin.HandlerFunc {
	return func(c *gin.Context) {

		symbol := c.Param("Symbol")
		exon_qcdata := c.Query("exon_qcdata"+symbol)

		exon := c.Query("exon"+symbol)
		exon_number, transcript, ok := strings.Cut(exon, " ")
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, "Could not find exon number or transcript")
			return
		}
		exon_float, err := strconv.ParseFloat(exon_number, 64)

		exonqc, err := db.ExonQc(symbol, exon_float, transcript)
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.String(http.StatusOK, "No data to plot")
			return
		}
		
		metric, ok := exonMetrics[exon_qcdata]
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Cannot retrieve the metrice data"})
			return
		}
		p, err := charts.GeneQCLineChart(
			exonqc.Exons, 
			exon_qcdata, 
			func(g *cleve.IncompleteExonsRunid) string { return g.RunId },
			metric,
		)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	
		s := p.RenderSnippet()
		c.String(http.StatusOK, s.Element+s.Script)

	}
} 