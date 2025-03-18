package interop

import (
	"testing"
)

func TestRunningAverage(t *testing.T) {
	testcases := []struct {
		name     string
		vals     []float64
		averages []float64
	}{
		{
			name:     "single value",
			vals:     []float64{1.0},
			averages: []float64{1.0},
		},
		{
			name:     "two values",
			vals:     []float64{1.0, 2.0},
			averages: []float64{1.0, 1.5},
		},
		{
			name:     "mix negative and positive",
			vals:     []float64{-5, 3, -10, 20, 13, -20},
			averages: []float64{-5, -1, -4, 2, 4.2, 0.1666666667},
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			ra := NewRunningAverage()
			for i, x := range c.vals {
				ra.Add(x)
				if ra.Average-c.averages[i] > 1e-6 {
					t.Errorf("expected average %.2f at %d, got %.2f", c.averages[i], i, ra.Average)
				}
			}
		})
	}
}
