package cleve

import (
	"slices"

	"github.com/gmc-norr/cleve/interop"
	"go.mongodb.org/mongo-driver/bson"
)

type Platforms struct {
	Platforms   []Platform
	isCondensed bool
}

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

func (p *Platforms) Add(platform Platform) {
	p.Platforms = append(p.Platforms, platform)
	if p.isCondensed {
		*p = p.Condense()
	}
}

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
	return Platforms{
		Platforms:   res,
		isCondensed: true,
	}
}

func (p Platforms) Names() []string {
	names := make([]string, 0, len(p.Platforms))
	for _, platform := range p.Platforms {
		names = append(names, platform.Name)
	}
	return names
}

type Platform struct {
	Name          string
	InstrumentIds []string
	Aliases       []string
	ReadyMarker   string
	RunCount      int
}

func NewPlatform(name, readyMarker string) *Platform {
	return &Platform{
		Name:        name,
		ReadyMarker: readyMarker,
	}
}

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
