package cleve

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	testcases := []struct {
		name     string
		string   string
		estring  string
		hasPatch bool
		error    bool
	}{
		{
			name:    "minor version",
			string:  "1.5",
			estring: "1.5",
		},
		{
			name:     "patch version",
			string:   "0.3.0",
			estring:  "0.3.0",
			hasPatch: true,
		},
		{
			name:     "patch version with trailing whitespace",
			string:   "1.0.0  ",
			estring:  "1.0.0",
			hasPatch: true,
		},
		{
			name:     "patch version with leading whitespace",
			string:   "\t5.0.3",
			estring:  "5.0.3",
			hasPatch: true,
		},
		{
			name:   "too many elements",
			string: "1.0.0.0",
			error:  true,
		},
		{
			name:   "too few elements",
			string: "1",
			error:  true,
		},
		{
			name:   "missing elements",
			string: "1.",
			error:  true,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			v, err := ParseVersion(c.string)
			if c.error != (err != nil) {
				if c.error {
					t.Errorf("expected error, got nil: %#v", v)
				} else {
					t.Errorf("expected no error, got this: %s", err.Error())
				}
			}
			if c.hasPatch != v.hasPatch {
				t.Errorf("expected hasPatch to be %t, got %t", c.hasPatch, v.hasPatch)
			}
		})
	}
}

func TestVersionEquality(t *testing.T) {
	testcases := []struct {
		name  string
		v1    string
		v2    string
		equal bool
	}{
		{
			name:  "equal minor version",
			v1:    "1.3",
			v2:    "1.3",
			equal: true,
		},
		{
			name:  "equal patch version",
			v1:    "1.3.10",
			v2:    "1.3.10",
			equal: true,
		},
		{
			name:  "inequal minor version",
			v1:    "1.3",
			v2:    "1.4",
			equal: false,
		},
		{
			name:  "inequal patch version",
			v1:    "1.3.1",
			v2:    "1.4.1",
			equal: false,
		},
		{
			name:  "compare patch and minor",
			v1:    "1.3.0",
			v2:    "1.3",
			equal: false,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			v1, _ := ParseVersion(c.v1)
			v2, _ := ParseVersion(c.v2)
			obs := v1.Equal(v2)
			if v1.Equal(v2) != c.equal {
				t.Errorf("expected equality to be %t, got %t", obs, c.equal)
			}
		})
	}
}

func TestVersionIsZero(t *testing.T) {
	testcases := []struct {
		name   string
		v      Version
		isZero bool
	}{
		{
			name:   "zero version",
			v:      Version{},
			isZero: true,
		},
		{
			name:   "zero version with haspatch",
			v:      Version{hasPatch: true},
			isZero: true,
		},
		{
			name:   "non-zero version",
			v:      Version{Major: 1, hasPatch: true},
			isZero: false,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			if c.v.IsZero() != c.isZero {
				t.Errorf("expected IsZero to be %t, got %t", c.isZero, c.v.IsZero())
			}
		})
	}
}
