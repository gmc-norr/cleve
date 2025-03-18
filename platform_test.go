package cleve

import (
	"testing"
)

func TestCondensePlatforms(t *testing.T) {
	testcases := []struct {
		name         string
		platforms    Platforms
		condensedLen int
		instruments  [][]string
		aliases      [][]string
	}{
		{
			name: "uncondensable",
			platforms: Platforms{
				Platforms: []Platform{
					{
						Name:          "platform 1",
						InstrumentIds: []string{"i1"},
						Aliases:       []string{"platform x"},
					},
					{
						Name:          "platform 2",
						InstrumentIds: []string{"i2"},
						Aliases:       []string{"platform y"},
					},
				},
			},
			condensedLen: 2,
			instruments:  [][]string{{"i1"}, {"i2"}},
			aliases:      [][]string{{"platform x"}, {"platform y"}},
		},
		{
			name: "condensable one platform",
			platforms: Platforms{
				Platforms: []Platform{
					{
						Name:          "platform 1",
						InstrumentIds: []string{"i1"},
					},
					{
						Name:          "platform 1",
						InstrumentIds: []string{"i2"},
					},
					{
						Name:          "platform 1",
						InstrumentIds: []string{"i3"},
					},
				},
			},
			condensedLen: 1,
			instruments:  [][]string{{"i1", "i2", "i3"}},
			aliases:      [][]string{{}},
		},
		{
			name: "condensable two platforms",
			platforms: Platforms{
				Platforms: []Platform{
					{
						Name:          "platform 1",
						InstrumentIds: []string{"i1"},
						Aliases:       []string{"platform x"},
					},
					{
						Name:          "platform 1",
						InstrumentIds: []string{"i2"},
					},
					{
						Name:          "platform 2",
						InstrumentIds: []string{"i3"},
						Aliases:       []string{"platform y"},
					},
				},
			},
			condensedLen: 2,
			instruments:  [][]string{{"i1", "i2"}, {"i3"}},
			aliases:      [][]string{{"platform x"}, {"platform y"}},
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			if c.platforms.isCondensed {
				t.Error("should not be condensed yet")
			}
			platforms := c.platforms.Condense()
			if !platforms.isCondensed {
				t.Error("platforms should be condensed")
			}
			if len(platforms.Platforms) != c.condensedLen {
				t.Errorf("expected length %d, got %d", c.condensedLen, len(platforms.Platforms))
			}
			for i, p := range platforms.Platforms {
				if len(p.InstrumentIds) != len(c.instruments[i]) {
					t.Errorf("expected %d instruments for %s, got %d", len(c.instruments[i]), p.Name, len(p.InstrumentIds))
				}
				t.Logf("%+v", c.aliases)
				if len(p.Aliases) != len(c.aliases[i]) {
					t.Errorf("expected %d aliases for %s, got %d", len(c.aliases[i]), p.Name, len(p.Aliases))
				}
			}
		})
	}
}
