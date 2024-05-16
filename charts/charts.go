package charts

import (
	"fmt"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/render"
)

type ChartData struct{}

type RunStats[T float64 | int] struct {
	Data  []RunStat[T]
	Label string
	Type  string
}

// Use a pointer for the value so that missing values
// are properly represented. Otherwise they would default
// to zero, and that's not right.
type RunStat[T float64 | int] struct {
	RunID string
	Value *T
}

func (s RunStats[T]) Plot() (render.Renderer, error) {
	switch s.Type {
	case "bar":
		return BarChart(s), nil
	case "line":
		return LineChart(s), nil
	default:
		return nil, fmt.Errorf("invalid chart type: %q", s.Type)
	}
}

func LineChart[T float64 | int](d RunStats[T]) *charts.Line {
	chart := charts.NewLine()
	chart.SetGlobalOptions(
		charts.WithTooltipOpts(opts.Tooltip{Show: true}),
	)
	xLabels := make([]string, 0)
	lineData := make([]opts.LineData, 0)
	for _, k := range d.Data {
		xLabels = append(xLabels, k.RunID)
		lineData = append(lineData, opts.LineData{Value: k.Value})
	}

	chart.SetXAxis(xLabels).
		AddSeries(d.Label, lineData, charts.WithLineChartOpts(
			opts.LineChart{ShowSymbol: true, SymbolSize: 5},
		))

	return chart
}

func BarChart[T float64 | int](d RunStats[T]) *charts.Bar {
	chart := charts.NewBar()
	chart.SetGlobalOptions(
		charts.WithTooltipOpts(opts.Tooltip{Show: true}),
	)
	xLabels := make([]string, 0)
	barData := make([]opts.BarData, 0)
	for _, k := range d.Data {
		xLabels = append(xLabels, k.RunID)
		barData = append(barData, opts.BarData{Value: k.Value})
	}

	chart.SetXAxis(xLabels).
		AddSeries(d.Label, barData)

	return chart
}
