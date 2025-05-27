package charts

import (
	"fmt"

	"github.com/gmc-norr/cleve/interop"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/render"
)

type RunStats[T interop.OptionalFloat | float64 | int] struct {
	Data   []RunStat[T]
	XLabel string
	YLabel string
	Label  string
	Type   string
}

// Use a pointer for the value so that missing values
// are properly represented. Otherwise they would default
// to zero, and that's not right.
type RunStat[T interop.OptionalFloat | float64 | int] struct {
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

type ScatterData struct {
	Data     []ScatterDatum
	XLabel   string
	YLabel   string
	Grouping string
}

type ScatterDatum struct {
	X     float64
	Y     float64
	Color any
}

func (d ScatterData) Plot() (render.Renderer, error) {
	return ScatterChart(d), nil
}

func LineChart[T interop.OptionalFloat | float64 | int](d RunStats[T]) *charts.Line {
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

func BarChart[T interop.OptionalFloat | float64 | int](d RunStats[T]) *charts.Bar {
	chart := charts.NewBar()
	chart.SetGlobalOptions(
		charts.WithTooltipOpts(opts.Tooltip{Show: true}),
		charts.WithXAxisOpts(opts.XAxis{Name: d.XLabel}),
		charts.WithYAxisOpts(opts.YAxis{Name: d.YLabel}),
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

func ScatterChart(d ScatterData) *charts.Scatter {
	chart := charts.NewScatter()
	chart.SetGlobalOptions(
		charts.WithLegendOpts(opts.Legend{Show: true}),
		charts.WithTooltipOpts(opts.Tooltip{Show: true}),
		charts.WithXAxisOpts(opts.XAxis{Name: d.XLabel}),
		charts.WithYAxisOpts(opts.YAxis{Name: d.YLabel}),
	)
	series := make(map[any][]opts.ScatterData)
	for _, k := range d.Data {
		sd := opts.ScatterData{
			Value:      []any{k.X, k.Y},
			Symbol:     "circle",
			SymbolSize: 5,
		}
		series[k.Color] = append(series[k.Color], sd)
	}
	chart.SetXAxis(nil)
	for k, v := range series {
		chart.AddSeries(fmt.Sprintf("%v", k), v)
	}
	return chart
}
