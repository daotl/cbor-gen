# cbor-gen

DAOT Labs' fork of [whyrusleeping/cbor-gen](https://github.com/whyrusleeping/cbor-gen).

Some basic utilities to generate fast path cbor codecs for your types.

## Usage

See `testgen/main.go`.

## New features of this fork

### Sort map keys according to RFC7049 & DAG-CBOR strict ordering

This adds proper RFC7049 map key sorting, to both bare maps and structs in map representation.
The other DAG-CBOR codec implementations strictly sort by length first, then bytes, as per
https://github.com/ipld/specs/blob/master/block-layer/codecs/dag-cbor.md#strictness.
The original [cbor-gen](https://github.com/whyrusleeping/cbor-gen) implementation will render 
CBOR that won't produce stable CIDs with a round-trip using another DAG-CBOR encoder.

For more details, see: https://github.com/whyrusleeping/cbor-gen/pull/42

### Embedded struct flattening

This fork additionally support an option `flattenEmbeddedStruct` in `WriteTupleEncodersToFile`
and `WriteMapEncodersToFile` functions to flatten embedded structs in serialization.

For example, we have following structs from `testing/flatten_tuple/types.go`:
```go
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

type StructValueFieldStruct struct {
	Signed  int64
	EmbeddedStruct EmbeddedStruct
	Binary  []byte
	NString NamedString
	Value   uint64
}

type StructPointerFieldStruct struct {
	Signed  int64
	EmbeddedStruct *EmbeddedStruct
	Binary  []byte
	NString NamedString
	Value   uint64
}

type EmbedByValueStruct struct {
	Signed  int64
	EmbeddedStruct
	Binary  []byte
	NString NamedString
	Value   uint64
}

type EmbedByPointerStruct struct {
	Signed  int64
	*EmbeddedStruct
	Binary  []byte
	NString NamedString
	Value   uint64
}
```

With the original [cbor-gen](https://github.com/whyrusleeping/cbor-gen), `StructValueFieldStruct`, 
`StructPointerFieldStruct`, `EmbedByValueStruct` and `EmbedByPointerStruct` will all be serialized
into the same byte sequence, which is different from that of `FlatStruct`.

With this fork and `flattenEmbeddedStruct` option set to `true`, `StructValueFieldStruct` and 
`StructPointerFieldStruct` will all be serialized into the same byte sequence as that of 
`FlatStruct`, which is different from that of `StructValueFieldStruct` and `StructPointerFieldStruct`.

The same flattening applies to recursively embedded structs with arbitrary levels of embedding.

> Note: For `flattenEmbeddedStruct` option to work, all structs recursively embedded by pointer
> (any levels) in the root struct need to generate their `InitNilEmbeddedStruct` methods with 
> `cbor-gen` and `flattenEmbeddedStruct` option also set to `true`, although their generated
> `MarshalCBOR` and `UnmarshalCBOR` methods won't be used when marshalling and unmarshalling the
> root struct. Alternatively, you can write the `InitNilEmbeddedStruct` methods by hand, see 
> `Inniter.InitNilEmbeddedStruct` in `utils.go` for description and `testing/flatten_tuple/cbor_gen.go`
> for examples.

```go
type Initter interface {
	// InitNilEmbeddedStruct prepare the struct for marshalling & unmarshalling by
	// recursively initialize the structs embedded by pointer to their zero values.
	InitNilEmbeddedStruct()
}
```

### Specifying struct field order

This fork additionally support an option `fieldOrder` in `WriteTupleEncodersToFile` to specify 
the orders of structs' fields. If specified (not nil), `fieldOrder` is a string slice of struct 
field names in the order they will be serialized, the remaining field names not found in the 
slice will be serialized after those specified in the original order of appearance in the code.

For example, we have the following two structs from `testing/flatten_tuple/types.go`:

```go
type FlatStruct struct {
	Signed  int64
	Foo     string
	Binary  []byte
	NString NamedString
	Value   uint64
}

type ReorderedFlatStruct struct {
	Foo     string
	Value   uint64
	Binary  []byte
	Signed  int64
	NString NamedString
}
```

They have the same fields with different orders. If we generate the `MarshalCBOR` and 
`UnmarshalCBOR` with the following options, they will be serialized into and deserialized
from the same byte sequence:

```go
cbg.WriteTupleEncodersToFile("testing/flatten_tuple/cbor_gen.go",
    "flatten_tuple", true, nil,
    flatten_tuple.FlatStruct{},
)

cbg.WriteTupleEncodersToFile("testing/flatten_tuple/cbor_gen_reordered.go",
    "flatten_tuple", true, []string{"Signed", "Foo", "Binary", "NString", "Value"},
    flatten_tuple.ReorderedFlatStruct{},
)
```

## License
MIT
