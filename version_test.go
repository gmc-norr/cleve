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
		string string
	}{
		{
			name:   "zero version",
			v:      Version{},
			isZero: true,
			string: "",
		},
		{
			name:   "zero version with haspatch",
			v:      Version{hasPatch: true},
			isZero: true,
			string: "",
		},
		{
			name:   "non-zero version",
			v:      Version{Major: 1, hasPatch: true},
			isZero: false,
			string: "1.0.0",
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			if c.v.IsZero() != c.isZero {
				t.Errorf("expected IsZero to be %t, got %t", c.isZero, c.v.IsZero())
			}
			if c.string != c.v.String() {
				t.Errorf("expected version string %q, got %q", c.string, c.v.String())
			}
		})
	}
}

func TestComparisons(t *testing.T) {
	testcases := []struct {
		name  string
		v1    Version
		v2    Version
		older bool
		newer bool
	}{
		{
			name:  "minor version v1 older",
			v1:    NewMinorVersion(1, 2),
			v2:    NewMinorVersion(1, 3),
			older: true,
			newer: false,
		},
		{
			name:  "minor version v1 newer",
			v1:    NewMinorVersion(2, 3),
			v2:    NewMinorVersion(1, 12),
			older: false,
			newer: true,
		},
		{
			name:  "patch version v1 older",
			v1:    NewPatchVersion(1, 2, 15),
			v2:    NewPatchVersion(1, 3, 2),
			older: true,
			newer: false,
		},
		{
			name:  "patch version v1 newer",
			v1:    NewPatchVersion(1, 2, 15),
			v2:    NewPatchVersion(1, 2, 2),
			older: false,
			newer: true,
		},
		{
			name:  "compare minor version with patch version",
			v1:    NewMinorVersion(2, 3),
			v2:    NewPatchVersion(1, 12, 0),
			older: false,
			newer: false,
		},
		{
			name:  "compare same version",
			v1:    NewPatchVersion(2, 3, 5),
			v2:    NewPatchVersion(2, 3, 5),
			older: false,
			newer: false,
		},
	}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			older := c.v1.OlderThan(c.v2)
			newer := c.v1.NewerThan(c.v2)
			if c.older != older {
				t.Errorf("expected older to be %t", c.older)
			}
			if c.newer != newer {
				t.Errorf("expected newer to be %t", c.newer)
			}
		})
	}
}
