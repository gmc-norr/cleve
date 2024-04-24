package interop

import (
	"fmt"
	"math"
)

type RunningVariance[T uint32 | uint64 | float32 | float64] struct {
	weighted bool
	wSum     float64
	w2Sum    float64
	Mean     float64
	s        float64
}

func NewRunningVariance[T uint32 | uint64 | float32 | float64](weighted bool) *RunningVariance[T] {
	return &RunningVariance[T]{weighted: weighted}
}

func (v *RunningVariance[T]) Push(x T, weight ...T) error {
	if math.IsNaN(float64(x)) {
		return nil
	}
	var w float64
	if !v.weighted {
		w = 1
		v.wSum += 1
		v.w2Sum += 1
	} else {
		if len(weight) != 1 {
			return fmt.Errorf("expected a single weight, got %d", len(weight))
		}
		w = float64(weight[0])
		v.wSum += w
		v.w2Sum += math.Pow(w, 2)
	}
	if v.wSum == 1 {
		v.Mean = float64(x)
		v.s = 0
	} else {
		oldMean := v.Mean
		v.Mean = oldMean + (w/v.wSum)*(float64(x)-oldMean)
		v.s = v.s + w*(float64(x)-oldMean)*(float64(x)-v.Mean)
	}
	return nil
}

func (v RunningVariance[T]) Var() float64 {
	if v.wSum > 1 {
		return v.s / (v.wSum - v.w2Sum/v.wSum)
	}
	return 0
}

func (v RunningVariance[T]) SD() float64 {
	return math.Sqrt(v.Var())
}
