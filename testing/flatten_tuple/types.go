package flatten_tuple

import (
	"github.com/daotl/cbor-gen/testing"
)

type NamedNumber = testing.NamedNumber
type NamedString = testing.NamedString
type SignedArray = testing.SignedArray
type SimpleTypeOne = testing.SimpleTypeOne
type SimpleTypeTwo = testing.SimpleTypeTwo
type SimpleTypeTree = testing.SimpleTypeTree

type EmbeddingStructOne struct {
	SimpleTypeOne
	*SimpleTypeTwo
	Foo          string
	Stuff        *SimpleTypeTwo
	Others       []uint64
	SignedOthers []int64
}

type EmbeddingStructTwo struct {
	SimpleTypeOne
	*EmbeddingStructOne
	Foo          string
	Value        uint64
	Stuff        *SimpleTypeTwo
	Others       []uint64
	SignedOthers []int64
	Test         [][]byte
	Dog          string
	Numbers      []NamedNumber
}

type EmbeddingStructThree struct {
	*EmbeddingStructTwo
	Foo          string
	Value        uint64
	Binary       []byte
	Stuff        *SimpleTypeTwo
	Others       []uint64
	SignedOthers []int64
	Test         [][]byte
	Dog          string
	Numbers      []NamedNumber
	Pizza        *uint64
	PointyPizza  *NamedNumber
	Arrrrrghay   [testing.Thingc]SimpleTypeOne
}

type FlatStruct struct {
	Signed  int64
	Foo     string
	Binary  []byte
	NString NamedString
	Value   uint64
}

type EmbeddedStruct struct {
	Value  uint64
	Foo    string
	Binary []byte
}

type EmbedByValueStruct struct {
	Signed int64
	EmbeddedStruct
	Binary  []byte
	NString NamedString
	Value   uint64
}

type EmbedByPointerStruct struct {
	Signed int64
	*EmbeddedStruct
	Binary  []byte
	NString NamedString
	Value   uint64
}
