package testing

import (
	cbg "github.com/bdware/cbor-gen"
)

const Thingc = 3

type NaturalNumber uint64

type SignedArray struct {
	Signed []uint64
}

type SimpleTypeOne struct {
	Foo    string
	Value  uint64
	Binary []byte
	Signed int64
}

type SimpleTypeTwo struct {
	Stuff        *SimpleTypeTwo
	Others       []uint64
	SignedOthers []int64
	Test         [][]byte
	Dog          string
	Numbers      []NaturalNumber
	Pizza        *uint64
	PointyPizza  *NaturalNumber
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

type DeferredContainer struct {
	Stuff    *SimpleTypeOne
	Deferred *cbg.Deferred
	Value    uint64
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
	Numbers      []NaturalNumber
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
	Numbers      []NaturalNumber
	Pizza        *uint64
	PointyPizza  *NaturalNumber
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
