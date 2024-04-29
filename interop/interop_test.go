package interop

import (
	"testing"
)

func TestReadConfig(t *testing.T) {
	r := ReadConfig{
		ReadLengths: map[int]int{
			1: 151,
			2: 8,
			3: 8,
			4: 151,
		},
	}

	cases := []struct {
		Cycle    int
		Expected int
	}{
		{0, -1},
		{1, 1},
		{151, 1},
		{152, 2},
		{159, 2},
		{160, 3},
		{167, 3},
		{168, 4},
		{318, 4},
		{319, -1},
	}

	for _, c := range cases {
		observed := r.CycleToRead(c.Cycle)
		if c.Expected != observed {
			t.Errorf("expected cycle %d to be read %d, not read %d", c.Cycle, c.Expected, observed)
		}
	}
}
