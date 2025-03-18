package cleve

import (
	"testing"
)

func TestCondensePlatforms(t *testing.T) {
	testcases := []struct {
		name        string
		platforms   Platforms
		names       []string
		instruments [][]string
		aliases     [][]string
	}{
		{
			name: "uncondensable",
			platforms: Platforms{
				Platforms: []Platform{
					{
						Name:          "platform 2",
						InstrumentIds: []string{"i2"},
						Aliases:       []string{"platform y"},
					},
					{
						Name:          "platform 1",
						InstrumentIds: []string{"i1"},
						Aliases:       []string{"platform x"},
					},
				},
			},
			names:       []string{"platform 1", "platform 2"},
			instruments: [][]string{{"i1"}, {"i2"}},
			aliases:     [][]string{{"platform x"}, {"platform y"}},
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
			names:       []string{"platform 1"},
			instruments: [][]string{{"i1", "i2", "i3"}},
			aliases:     [][]string{{}},
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
			names:       []string{"platform 1", "platform 2"},
			instruments: [][]string{{"i1", "i2"}, {"i3"}},
			aliases:     [][]string{{"platform x"}, {"platform y"}},
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
			if len(platforms.Platforms) != len(c.names) {
				t.Errorf("expected length %d, got %d", len(c.names), len(platforms.Platforms))
			}
			for i, p := range platforms.Platforms {
				if p.Name != c.names[i] {
					t.Errorf("expected platform %d to be %s, got %s", i, c.names[i], p.Name)
				}
				if len(p.InstrumentIds) != len(c.instruments[i]) {
					t.Errorf("expected %d instruments for %s, got %d", len(c.instruments[i]), p.Name, len(p.InstrumentIds))
				}
				if len(p.Aliases) != len(c.aliases[i]) {
					t.Errorf("expected %d aliases for %s, got %d", len(c.aliases[i]), p.Name, len(p.Aliases))
				}
			}
		})
	}
}
