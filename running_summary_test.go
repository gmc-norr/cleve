package cleve

import (
	"math"
	"testing"
)

func almostEqual[T float32 | float64](a, b T) bool {
	return math.Abs(float64(a-b)) < math.Pow(10, -6)
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}

func TestRunningVariance(t *testing.T) {
	var numbers = []float32{
		-0.2386511,
		-0.8291323,
		0.482924,
		-1.083026,
		-0.02429886,
		1.09931,
		0.6133231,
		-0.9356931,
		2.407577,
		-1.656025,
	}
	mean := -0.01636933
	variance := 1.460834
	sd := 1.208649

	v := RunningSummary[float32]{}
	for _, x := range numbers {
		v.Push(x)
	}

	if !almostEqual(v.Mean, mean) {
		t.Fatalf("expected mean %f, got %f", mean, v.Mean)
	}

	if !almostEqual(v.Var(), variance) {
		t.Fatalf("expected variance %f, got %f", variance, v.Var())
	}

	if !almostEqual(v.SD(), sd) {
		t.Fatalf("expected sd %f, got %f", sd, v.SD())
	}
}
