package testing

import (
	"github.com/ipfs/go-cid"

	cbg "github.com/bdware/cbor-gen"
)

const Thingc = 3

type NamedNumber uint64
type NamedString string

type SignedArray struct {
	Signed []uint64
}

type SimpleTypeOne struct {
	Foo     string
	Value   uint64
	Binary  []byte
	Signed  int64
	NString NamedString
}

type SimpleTypeTwo struct {
	Stuff        *SimpleTypeTwo
	Others       []uint64
	SignedOthers []int64
	Test         [][]byte
	Dog          string
	Numbers      []NamedNumber
	Pizza        *uint64
	PointyPizza  *NamedNumber
	Arrrrrghay   [Thingc]SimpleTypeOne
}

type SimpleTypeTree struct {
	Stuff                            *SimpleTypeTree
	Stufff                           *SimpleTypeTwo
	Others                           []uint64
	Test                             [][]byte
	Dog                              string
	SixtyThreeBitIntegerWithASignBit int64
	NotPizza                         *uint64
}

type SimpleStructV1 struct {
	OldStr    string
	OldBytes  []byte
	OldNum    uint64
	OldPtr    *cid.Cid
	OldMap    map[string]SimpleTypeOne
	OldArray  []SimpleTypeOne
	OldStruct SimpleTypeOne
}

type SimpleStructV2 struct {
	OldStr string
	NewStr string

	OldBytes []byte
	NewBytes []byte

	OldNum uint64
	NewNum uint64

	OldPtr *cid.Cid
	NewPtr *cid.Cid

	OldMap map[string]SimpleTypeOne
	NewMap map[string]SimpleTypeOne

	OldArray []SimpleTypeOne
	NewArray []SimpleTypeOne

	OldStruct SimpleTypeOne
	NewStruct SimpleTypeOne
}

type DeferredContainer struct {
	Stuff    *SimpleTypeOne
	Deferred *cbg.Deferred
	Value    uint64
}

type FixedArrays struct {
	Bytes  [20]byte
	Uint8  [20]uint8
	Uint64 [20]uint64
}

type ThingWithSomeTime struct {
	When    cbg.CborTime
	Stuff   int64
	CatName string
}

// Do not add fields to this type.
type NeedScratchForMap struct {
	Thing bool
}

type EmbeddingAnonymousStructOne struct {
	SimpleTypeOne
	*SimpleTypeTwo
	Foo          string
	Stuff        *SimpleTypeTwo
	Others       []uint64
	SignedOthers []int64
}

func (s EmbeddingAnonymousStructOne) Zero() {
	s.SimpleTypeTwo = &SimpleTypeTwo{}
}

type EmbeddingAnonymousStructTwo struct {
	SimpleTypeOne
	*EmbeddingAnonymousStructOne
	Foo          string
	Value        uint64
	Stuff        *SimpleTypeTwo
	Others       []uint64
	SignedOthers []int64
	Test         [][]byte
	Dog          string
	Numbers      []NamedNumber
}

func (s EmbeddingAnonymousStructTwo) Zero() {
	s.EmbeddingAnonymousStructOne = &EmbeddingAnonymousStructOne{}
	s.EmbeddingAnonymousStructOne.Zero()
}

type EmbeddingAnonymousStructThree struct {
	*EmbeddingAnonymousStructTwo
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
	Arrrrrghay   [Thingc]SimpleTypeOne
}

func (s EmbeddingAnonymousStructThree) Zero() {
	s.EmbeddingAnonymousStructTwo = &EmbeddingAnonymousStructTwo{}
	s.EmbeddingAnonymousStructTwo.Zero()
}

type EmbeddingAnonymousStructTree struct {
	Stuff  *EmbeddingAnonymousStructTree
	Stufff *EmbeddingAnonymousStructOne
	*EmbeddingAnonymousStructTwo
	Others                           []uint64
	Test                             [][]byte
	Dog                              string
	SixtyThreeBitIntegerWithASignBit int64
	NotPizza                         *uint64
}

func (s EmbeddingAnonymousStructTree) Zero() {
	s.EmbeddingAnonymousStructTwo = &EmbeddingAnonymousStructTwo{}
	s.EmbeddingAnonymousStructTwo.Zero()
}
