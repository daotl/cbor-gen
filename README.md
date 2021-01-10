# cbor-gen

Fork of [cbor-gen](https://github.com/bdware/cbor-gen) used by BDWare projects.

Some basic utilities to generate fast path cbor codecs for your types.

New features of this fork:
- [Correctly marshal/unmarshal struct fields of slice type with nil values to/from CborNull](https://github.com/whyrusleeping/cbor-gen/pull/27)
- Flatten embedded anonymous struct fields 
  
  For example, if we have the following structs:
  ```go
  type FlatStruct struct {
      Foo     string
      Value   uint64
      Binary  []byte
      Signed  int64
      NString NamedString
  }  

  type EmbeddedStruct struct {
      Foo     string
      Value   uint64
      Binary  []byte
  }
  
  type ExplicitEmbeddingStruct struct {
      EmbeddedStruct EmbeddedStruct
      Binary  []byte
      Signed  int64
      NString NamedString
  }
  
  type ExplicitEmbeddingStruct2 struct {
      EmbeddedStruct *EmbeddedStruct
      Binary  []byte
      Signed  int64
      NString NamedString
  }
  
  type AnonymousEmbeddingStruct struct {
      EmbeddedStruct
      Binary  []byte
      Signed  int64
      NString NamedString
  }

  type AnonymousEmbeddingStruct2 struct {
      *EmbeddedStruct
      Binary  []byte
      Signed  int64
      NString NamedString
  }
  ```
  
  With the original [cbor-gen](https://github.com/whyrusleeping/cbor-gen), `ExplicitEmbeddingStruct`, `ExplicitEmbeddingStruct2`, `AnonymousEmbeddingStruct` and `AnonymousEmbeddingStruct2` will all be serialized into the same byte sequence, which is different from that of `FlatStruct`.

  With this fork, `AnonymousEmbeddingStruct` and `AnonymousEmbeddingStruct2` will be serialized into the same byte sequence as that of `FlatStruct`, which is different from that of `ExplicitEmbeddingStruct` and `ExplicitEmbeddingStruct2`.

  The same flattening applies to structs with arbitrary levels of anonymous embedding. 

  > Note: Before calling `UnmarshalCBOR` on an embedding struct `obj`, you must make sure that all anonymous structs directly and indirectly (multiple levels of embedding) embedded in `obj` by pointers are initialized (not null).

## License
MIT
