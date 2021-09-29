package main

import (
	cbg "github.com/daotl/cbor-gen"
	types "github.com/daotl/cbor-gen/testing"
	"github.com/daotl/cbor-gen/testing/flatten_map"
	"github.com/daotl/cbor-gen/testing/flatten_tuple"
	"github.com/daotl/cbor-gen/testing/noflatten_map"
	"github.com/daotl/cbor-gen/testing/noflatten_tuple"
)

func main() {
	if err := cbg.WriteTupleEncodersToFile("testing/cbor_gen.go",
		"testing", false, nil,
		types.SignedArray{},
		types.SimpleTypeOne{},
		types.SimpleTypeTwo{},
		types.DeferredContainer{},
		types.FixedArrays{},
		types.ThingWithSomeTime{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile("testing/cbor_map_gen.go",
		"testing", false,
		types.SimpleTypeTree{},
		types.NeedScratchForMap{},
		types.SimpleStructV1{},
		types.SimpleStructV2{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteTupleEncodersToFile("testing/noflatten_tuple/cbor_gen.go",
		"noflatten_tuple", false, nil,
		noflatten_tuple.EmbeddingStructOne{},
		noflatten_tuple.EmbeddingStructTwo{},
		noflatten_tuple.EmbeddingStructThree{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile("testing/noflatten_map/cbor_gen.go",
		"noflatten_map", false,
		noflatten_map.EmbeddingStructOne{},
		noflatten_map.EmbeddingStructTwo{},
		noflatten_map.EmbeddingStructThree{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteTupleEncodersToFile("testing/flatten_tuple/cbor_gen.go",
		"flatten_tuple", true, nil,
		flatten_tuple.EmbeddingStructOne{},
		flatten_tuple.EmbeddingStructTwo{},
		flatten_tuple.EmbeddingStructThree{},
		flatten_tuple.FlatStruct{},
		flatten_tuple.EmbeddedStruct{},
		flatten_tuple.EmbedByValueStruct{},
		flatten_tuple.EmbedByPointerStruct{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteTupleEncodersToFile("testing/flatten_tuple/cbor_gen_reordered.go",
		"flatten_tuple", true, []string{"Signed", "Foo", "Binary", "NString", "Value"},
		flatten_tuple.ReorderedFlatStruct{},
		flatten_tuple.ReorderedEmbedByValueStruct{},
		flatten_tuple.ReorderedEmbedByPointerStruct{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile("testing/flatten_map/cbor_gen.go",
		"flatten_map", true,
		flatten_map.EmbeddingStructOne{},
		flatten_map.EmbeddingStructTwo{},
		flatten_map.EmbeddingStructThree{},
		flatten_tuple.FlatStruct{},
		flatten_tuple.EmbeddedStruct{},
		flatten_tuple.EmbedByValueStruct{},
		flatten_tuple.EmbedByPointerStruct{},
	); err != nil {
		panic(err)
	}
}
