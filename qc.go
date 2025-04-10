package cleve

import (
	"github.com/gmc-norr/cleve/interop"
)

type QcResult struct {
	PaginationMetadata `bson:"metadata" json:"metadata"`
	InteropSummary     []interop.InteropSummary `bson:"interop" json:"interop"`
}
