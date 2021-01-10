package main

import (
	cbg "github.com/bdware/cbor-gen"
	types "github.com/bdware/cbor-gen/testing"
)

func main() {
	if err := cbg.WriteTupleEncodersToFile("testing/cbor_gen.go", "testing",
		types.SignedArray{},
		types.SimpleTypeOne{},
		types.SimpleTypeTwo{},
		types.DeferredContainer{},
		types.EmbeddingAnonymousStructOne{},
		types.EmbeddingAnonymousStructTwo{},
		types.EmbeddingAnonymousStructThree{},
	); err != nil {
		panic(err)
	}

	if err := cbg.WriteMapEncodersToFile("testing/cbor_map_gen.go", "testing",
		types.SimpleTypeTree{},
		types.EmbeddingAnonymousStructTree{},
	); err != nil {
		panic(err)
	}
}
