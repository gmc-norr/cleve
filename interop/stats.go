package interop

type RunningAverage struct {
	Average float64
	count   int
}

func NewRunningAverage() RunningAverage {
	return RunningAverage{}
}

func (r *RunningAverage) Add(val float64) {
	r.count++
	r.Average = r.Average*(float64(r.count-1)/float64(r.count)) + val/float64(r.count)
}
