package cleve

import (
	"slices"
	"strings"

	"github.com/gmc-norr/cleve/interop"
	"go.mongodb.org/mongo-driver/bson"
)

// Platforms represents a collection of sequencing platforms.
type Platforms struct {
	Platforms   []Platform
	isCondensed bool
}

// Get retrieves a platform by name. If a platform with that name is found,
// the platform is returned with ok set to true. Otherwise a default struct
// for the platform is returned with ok set to false.
func (p Platforms) Get(name string) (platform Platform, ok bool) {
	var match bool
	if !p.isCondensed {
		p = p.Condense()
	}
	for _, pl := range p.Platforms {
		match = pl.Name == name
		if !match {
			match = slices.Contains(pl.Aliases, name)
		}
		if match {
			return pl, true
		}
	}
	return platform, false
}

// Add adds a platform to the collection.
func (p *Platforms) Add(platform Platform) {
	p.Platforms = append(p.Platforms, platform)
	if p.isCondensed {
		*p = p.Condense()
	}
}

// Condense merges platforms by name. This collects all the instrument IDs and aliases
// for platforms with the same name, as well as sums up the run count for all individual
// instruments. The function will leave the original object intact and return a new
// Platforms object. Platforms will be sorted lexicographically by name.
func (p Platforms) Condense() Platforms {
	condensed := make(map[string]Platform, 0)
	for _, platform := range p.Platforms {
		resP := condensed[platform.Name]
		resP.Name = platform.Name
		resP.InstrumentIds = append(resP.InstrumentIds, platform.InstrumentIds...)
		resP.Aliases = append(resP.Aliases, platform.Aliases...)
		resP.RunCount += platform.RunCount
		condensed[platform.Name] = resP
	}
	res := make([]Platform, 0)
	for _, v := range condensed {
		slices.Sort(v.Aliases)
		v.Aliases = slices.Compact(v.Aliases)
		res = append(res, v)
	}
	slices.SortFunc(res, func(a Platform, b Platform) int {
		return strings.Compare(a.Name, b.Name)
	})
	return Platforms{
		Platforms:   res,
		isCondensed: true,
	}
}

// Names returns a slice of platform names that are present in the collection.
// If the platforms have not been condensed there might be duplicates in the
// resulting slice.
func (p Platforms) Names() []string {
	names := make([]string, 0, len(p.Platforms))
	for _, platform := range p.Platforms {
		names = append(names, platform.Name)
	}
	return names
}

// Platform represents a sequencing platform.
type Platform struct {
	Name          string
	InstrumentIds []string
	Aliases       []string
	ReadyMarker   string
	RunCount      int
}

// UnmarshalBSON unmarshals a BSON representation into a Platform struct.
func (p *Platform) UnmarshalBSON(data []byte) error {
	type PlatformAlias struct {
		InstrumentId string   `bson:"instrument_id"`
		Aliases      []string `bson:"aliases"`
		Count        int      `bson:"count"`
	}
	var platformAlias PlatformAlias
	err := bson.Unmarshal(data, &platformAlias)
	if err != nil {
		return err
	}
	p.Name = interop.IdentifyPlatform(platformAlias.InstrumentId)
	p.ReadyMarker = interop.PlatformReadyMarker(p.Name)
	for _, a := range platformAlias.Aliases {
		if a != p.Name {
			p.Aliases = append(p.Aliases, a)
		}
	}
	p.InstrumentIds = []string{platformAlias.InstrumentId}
	p.RunCount = platformAlias.Count
	return nil
}
