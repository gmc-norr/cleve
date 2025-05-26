package cleve

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	Major    int
	Minor    int
	Patch    int
	hasPatch bool
}

func (v Version) HasPatch() bool {
	return v.hasPatch
}

func (v Version) IsZero() bool {
	return v.Major == 0 && v.Minor == 0 && v.Patch == 0
}

// OlderThan checks if version `v` is older than verson `other`. Only returns
// `true` if `v` is strictly smaller than `other`. If the versions are identical
// `false` is returned. If one of the versions has a patch version number and the
// other doesn't, false is returned.
func (v Version) OlderThan(other Version) bool {
	if v.hasPatch != other.hasPatch {
		return false
	}
	if v.Major < other.Major {
		return true
	} else if v.Major != other.Major {
		return false
	}
	if v.Minor < other.Minor {
		return true
	} else if v.Minor != other.Minor {
		return false
	}
	if v.hasPatch && other.hasPatch && v.Patch < other.Patch {
		return true
	}
	return false
}

// NewerThan checks if version `v` is newer than verson `other`. Only returns
// `true` if `v` is strictly larger than `other`. If the versions are identical
// `false` is returned. If one of the versions has a patch version number and the
// other doesn't, false is returned.
func (v Version) NewerThan(other Version) bool {
	if v.hasPatch != other.hasPatch {
		return false
	}
	if v.Major > other.Major {
		return true
	} else if v.Major != other.Major {
		return false
	}
	if v.Minor > other.Minor {
		return true
	} else if v.Minor != other.Minor {
		return false
	}
	if v.hasPatch && other.hasPatch && v.Patch > other.Patch {
		return true
	}
	return false
}

func (v Version) Equal(other Version) bool {
	return v.Major == other.Major &&
		v.Minor == other.Minor &&
		v.Patch == other.Patch &&
		v.hasPatch == other.hasPatch
}

func (v Version) String() string {
	if v.IsZero() {
		return ""
	}
	if v.hasPatch {
		return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	}
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

func (v Version) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(v.String())), nil
}

func NewMinorVersion(major int, minor int) Version {
	return Version{
		Major:    major,
		Minor:    minor,
		hasPatch: false,
	}
}

func NewPatchVersion(major int, minor int, patch int) Version {
	return Version{
		Major:    major,
		Minor:    minor,
		Patch:    patch,
		hasPatch: true,
	}
}

func ParseVersion(vs string) (Version, error) {
	v := Version{
		hasPatch: false,
	}
	elems := strings.Split(strings.TrimSpace(vs), ".")
	if len(elems) < 2 || len(elems) > 3 {
		return v, fmt.Errorf("invalid format for version: %q", vs)
	}
	nums := make([]int, len(elems))
	for i := range len(elems) {
		n, err := strconv.ParseUint(elems[i], 10, 32)
		if err != nil {
			return v, err
		}
		nums[i] = int(n)
	}
	switch len(nums) {
	case 2:
		v = NewMinorVersion(nums[0], nums[1])
	case 3:
		v = NewPatchVersion(nums[0], nums[1], nums[2])
	default:
		return v, fmt.Errorf("invalid number of elements in version: %v", nums)
	}
	return v, nil
}
