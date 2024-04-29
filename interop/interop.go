package interop

import (
	"encoding/binary"
	"fmt"
	"io"
	"sort"
)

type InteropHeader struct {
	Version    uint8
	RecordSize uint8
}

func (h *InteropHeader) Parse(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, h); err != nil {
		return fmt.Errorf("%s when parsing interop header", err.Error())
	}
	return nil
}

func (h InteropHeader) GetVersion() uint8 {
	return h.Version
}

func (h InteropHeader) GetRecordSize() uint8 {
	return h.RecordSize
}

type InteropFile interface {
	GetVersion() uint8
	GetRecordSize() uint8
}

type InteropRecord interface {
	Type() string
}

type InteropRecordHolder interface {
	Records() []InteropRecord
}

type Tile interface {
	Parse(io.Reader) error
}
type Tile16 uint16
type Tile32 uint32

func (t *Tile16) Parse(r io.Reader) error {
	return binary.Read(r, binary.LittleEndian, t)
}

func (t *Tile32) Parse(r io.Reader) error {
	return binary.Read(r, binary.LittleEndian, t)
}

type ReadConfig struct {
	ReadLengths map[int]int
}

func (r ReadConfig) CycleToRead(cycle int) int {
	reads := make([]int, 0)
	for key := range r.ReadLengths {
		reads = append(reads, key)
	}
	sort.Ints(reads)

	currentStart := 0
	for _, read := range reads {
		if cycle > currentStart && cycle <= currentStart + r.ReadLengths[read] {
			return read
		}
		currentStart += r.ReadLengths[read]
	}
	return -1
}

type LaneStats = map[int]*RunningSummary[float64]
type TileStats = map[int]map[int]*RunningSummary[float64]
type ReadStats = map[int]map[int]*RunningSummary[float64]
type CycleStats = map[int]map[int]map[int]*RunningSummary[float64]
