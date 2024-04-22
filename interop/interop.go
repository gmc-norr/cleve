package interop

import (
	"encoding/binary"
	"io"
)

type InteropFile struct {
	Version uint8
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
