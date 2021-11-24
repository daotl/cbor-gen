package testing_test

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ipfs/go-cid"

	cbg "github.com/daotl/cbor-gen"
	types "github.com/daotl/cbor-gen/testing"
	"github.com/daotl/cbor-gen/testing/flatten_map"
	"github.com/daotl/cbor-gen/testing/flatten_tuple"
	"github.com/daotl/cbor-gen/testing/noflatten_map"
	"github.com/daotl/cbor-gen/testing/noflatten_tuple"
)

var alwaysEqual = cmp.Comparer(func(_, _ interface{}) bool { return true })

// This option handles slices and maps of any type.
var alwaysEqualOpt = cmp.FilterValues(func(x, y interface{}) bool {
	vx, vy := reflect.ValueOf(x), reflect.ValueOf(y)
	return (vx.IsValid() && vy.IsValid() && vx.Type() == vy.Type()) &&
		(vx.Kind() == reflect.Slice || vx.Kind() == reflect.Map) &&
		(vx.Len() == 0 && vy.Len() == 0)
}, alwaysEqual)

func TestSimpleSigned(t *testing.T) {
	testTypeRoundtrips(t, reflect.TypeOf(types.SignedArray{}), false)
}

func TestSimpleTypeOne(t *testing.T) {
	testTypeRoundtrips(t, reflect.TypeOf(types.SimpleTypeOne{}), false)
}

func TestSimpleTypeTwo(t *testing.T) {
	testTypeRoundtrips(t, reflect.TypeOf(types.SimpleTypeTwo{}), false)
}

func TestSimpleTypeTree(t *testing.T) {
	testTypeRoundtrips(t, reflect.TypeOf(types.SimpleTypeTree{}), false)
}

func TestNeedScratchForMap(t *testing.T) {
	testTypeRoundtrips(t, reflect.TypeOf(types.NeedScratchForMap{}), false)
}

func TestNoFlattenTuple(t *testing.T) {
	testTypeRoundtrips(t, reflect.TypeOf(noflatten_tuple.EmbeddingStructOne{}), false)
	testTypeRoundtrips(t, reflect.TypeOf(noflatten_tuple.EmbeddingStructTwo{}), false)
	testTypeRoundtrips(t, reflect.TypeOf(noflatten_tuple.EmbeddingStructThree{}), false)
}

func TestNoFlattenMap(t *testing.T) {
	testTypeRoundtrips(t, reflect.TypeOf(noflatten_map.EmbeddingStructOne{}), false)
	testTypeRoundtrips(t, reflect.TypeOf(noflatten_map.EmbeddingStructTwo{}), false)
	testTypeRoundtrips(t, reflect.TypeOf(noflatten_map.EmbeddingStructThree{}), false)
}

func TestFlattenTuple(t *testing.T) {
	testTypeRoundtrips(t, reflect.TypeOf(flatten_tuple.EmbeddingStructOne{}), true)
	testTypeRoundtrips(t, reflect.TypeOf(flatten_tuple.EmbeddingStructTwo{}), true)
	testTypeRoundtrips(t, reflect.TypeOf(flatten_tuple.EmbeddingStructThree{}), true)
}

func TestFlattenMap(t *testing.T) {
	testTypeRoundtrips(t, reflect.TypeOf(flatten_map.EmbeddingStructOne{}), true)
	testTypeRoundtrips(t, reflect.TypeOf(flatten_map.EmbeddingStructTwo{}), true)
	testTypeRoundtrips(t, reflect.TypeOf(flatten_map.EmbeddingStructThree{}), true)
}

func TestFlattenEmbeddedStruct(t *testing.T) {
	r := rand.New(rand.NewSource(56887))

	// Test flatten_tuple
	for i := 0; i < 1000; i++ {
		val, ok := quick.Value(reflect.TypeOf(flatten_tuple.FlatStruct{}), r)
		if !ok {
			t.Fatal("failed to generate test value")
		}

		obj := val.Addr().Interface().(cbg.CBORMarshaler)
		buf := new(bytes.Buffer)
		if err := obj.MarshalCBOR(buf); err != nil {
			t.Fatal("i guess its fine to fail marshaling")
		}
		flatenc := buf.Bytes()

		sv := &flatten_tuple.EmbedByValueStruct{}
		if err := sv.UnmarshalCBOR(bytes.NewReader(flatenc)); err != nil {
			t.Logf("got bad bytes: %x", flatenc)
			t.Fatal("failed to unmarshal object: ", err)
		}
		buf = new(bytes.Buffer)
		if err := sv.MarshalCBOR(buf); err != nil {
			t.Fatal("i guess its fine to fail marshaling")
		}
		enc := buf.Bytes()

		if !bytes.Equal(enc, flatenc) {
			t.Fatalf("objects encodings different: %x != %x", enc, flatenc)
		}

		sp := &flatten_tuple.EmbedByPointerStruct{}
		if err := sp.UnmarshalCBOR(bytes.NewReader(flatenc)); err != nil {
			t.Logf("got bad bytes: %x", flatenc)
			t.Fatal("failed to unmarshal object: ", err)
		}
		buf = new(bytes.Buffer)
		if err := sp.MarshalCBOR(buf); err != nil {
			t.Fatal("i guess its fine to fail marshaling")
		}
		enc = buf.Bytes()

		if !bytes.Equal(enc, flatenc) {
			t.Fatalf("objects encodings different: %x != %x", enc, flatenc)
		}

		sr := &flatten_tuple.ReorderedFlatStruct{}
		if err := sr.UnmarshalCBOR(bytes.NewReader(flatenc)); err != nil {
			t.Logf("got bad bytes: %x", flatenc)
			t.Fatal("failed to unmarshal object: ", err)
		}
		buf = new(bytes.Buffer)
		if err := sr.MarshalCBOR(buf); err != nil {
			t.Fatal("i guess its fine to fail marshaling")
		}
		enc = buf.Bytes()

		if !bytes.Equal(enc, flatenc) {
			t.Fatalf("objects encodings different: %x != %x", enc, flatenc)
		}

		srv := &flatten_tuple.ReorderedEmbedByValueStruct{}
		if err := srv.UnmarshalCBOR(bytes.NewReader(flatenc)); err != nil {
			t.Logf("got bad bytes: %x", flatenc)
			t.Fatal("failed to unmarshal object: ", err)
		}
		buf = new(bytes.Buffer)
		if err := srv.MarshalCBOR(buf); err != nil {
			t.Fatal("i guess its fine to fail marshaling")
		}
		enc = buf.Bytes()

		if !bytes.Equal(enc, flatenc) {
			t.Fatalf("objects encodings different: %x != %x", enc, flatenc)
		}

		srp := &flatten_tuple.ReorderedEmbedByValueStruct{}
		if err := srp.UnmarshalCBOR(bytes.NewReader(flatenc)); err != nil {
			t.Logf("got bad bytes: %x", flatenc)
			t.Fatal("failed to unmarshal object: ", err)
		}
		buf = new(bytes.Buffer)
		if err := srp.MarshalCBOR(buf); err != nil {
			t.Fatal("i guess its fine to fail marshaling")
		}
		enc = buf.Bytes()

		if !bytes.Equal(enc, flatenc) {
			t.Fatalf("objects encodings different: %x != %x", enc, flatenc)
		}
	}

	// Test flatten_map
	for i := 0; i < 1000; i++ {
		val, ok := quick.Value(reflect.TypeOf(flatten_map.FlatStruct{}), r)
		if !ok {
			t.Fatal("failed to generate test value")
		}

		obj := val.Addr().Interface().(cbg.CBORMarshaler)
		buf := new(bytes.Buffer)
		if err := obj.MarshalCBOR(buf); err != nil {
			t.Fatal("i guess its fine to fail marshaling")
		}
		flatenc := buf.Bytes()

		sv := &flatten_map.EmbedByValueStruct{}
		if err := sv.UnmarshalCBOR(bytes.NewReader(flatenc)); err != nil {
			t.Logf("got bad bytes: %x", flatenc)
			t.Fatal("failed to unmarshal object: ", err)
		}
		buf = new(bytes.Buffer)
		if err := sv.MarshalCBOR(buf); err != nil {
			t.Fatal("i guess its fine to fail marshaling")
		}
		enc := buf.Bytes()

		if !bytes.Equal(enc, flatenc) {
			t.Fatalf("objects encodings different: %x != %x", enc, flatenc)
		}

		sp := &flatten_map.EmbedByPointerStruct{}
		if err := sp.UnmarshalCBOR(bytes.NewReader(flatenc)); err != nil {
			t.Logf("got bad bytes: %x", flatenc)
			t.Fatal("failed to unmarshal object: ", err)
			t.Fatal("failed to sunmarshal object: ", err)
		}
		buf = new(bytes.Buffer)
		if err := sp.MarshalCBOR(buf); err != nil {
			t.Fatal("i guess its fine to fail marshaling")
		}
		enc = buf.Bytes()

		if !bytes.Equal(enc, flatenc) {
			t.Fatalf("objects encodings different: %x != %x", enc, flatenc)
		}
	}
}

func testValueRoundtrip(t *testing.T, obj cbg.CBORMarshaler, nobj cbg.CBORUnmarshaler,
	onlyCompareBytes bool) {

	buf := new(bytes.Buffer)
	if err := obj.MarshalCBOR(buf); err != nil {
		t.Fatal("i guess its fine to fail marshaling")
	}

	enc := buf.Bytes()

	if err := nobj.UnmarshalCBOR(bytes.NewReader(enc)); err != nil {
		t.Logf("got bad bytes: %x", enc)
		t.Fatal("failed to round trip object: ", err)
	}

	if !onlyCompareBytes {
		if !cmp.Equal(obj, nobj, alwaysEqualOpt) {
			t.Logf("%#v != %#v", obj, nobj)
			t.Log("not equal after round trip!")
		}
	}

	nbuf := new(bytes.Buffer)
	if err := nobj.(cbg.CBORMarshaler).MarshalCBOR(nbuf); err != nil {
		t.Fatal("failed to remarshal object: ", err)
	}

	if !bytes.Equal(nbuf.Bytes(), enc) {
		t.Fatalf("objects encodings different: %x != %x", nbuf.Bytes(), enc)
	}

}

func testTypeRoundtrips(t *testing.T, typ reflect.Type, onlyCompareBytes bool) {
	r := rand.New(rand.NewSource(56887))
	for i := 0; i < 1000; i++ {
		val, ok := quick.Value(typ, r)
		if !ok {
			t.Fatal("failed to generate test value")
		}

		obj := val.Addr().Interface().(cbg.CBORMarshaler)
		nobj := reflect.New(typ).Interface().(cbg.CBORUnmarshaler)
		testValueRoundtrip(t, obj, nobj, onlyCompareBytes)
	}
}

func TestDeferredContainer(t *testing.T) {
	zero := &types.DeferredContainer{}
	recepticle := &types.DeferredContainer{}
	testValueRoundtrip(t, zero, recepticle, false)
}

func TestNilValueDeferredUnmarshaling(t *testing.T) {
	var zero types.DeferredContainer
	zero.Deferred = &cbg.Deferred{Raw: []byte{0xf6}}

	buf := new(bytes.Buffer)
	if err := zero.MarshalCBOR(buf); err != nil {
		t.Fatal(err)
	}

	var n types.DeferredContainer
	if err := n.UnmarshalCBOR(buf); err != nil {
		t.Fatal(err)
	}

	if n.Deferred == nil {
		t.Fatal("shouldnt be nil!")
	}
}

func TestFixedArrays(t *testing.T) {
	zero := &types.FixedArrays{}
	recepticle := &types.FixedArrays{}
	testValueRoundtrip(t, zero, recepticle, false)
}

func TestTimeIsh(t *testing.T) {
	val := &types.ThingWithSomeTime{
		When:    cbg.CborTime(time.Now()),
		Stuff:   1234,
		CatName: "hank",
	}

	buf := new(bytes.Buffer)
	if err := val.MarshalCBOR(buf); err != nil {
		t.Fatal(err)
	}

	out := types.ThingWithSomeTime{}
	if err := out.UnmarshalCBOR(buf); err != nil {
		t.Fatal(err)
	}

	if out.When.Time().UnixNano() != val.When.Time().UnixNano() {
		t.Fatal("time didnt round trip properly", out.When.Time(), val.When.Time())
	}

	if out.Stuff != val.Stuff {
		t.Fatal("no")
	}

	if out.CatName != val.CatName {
		t.Fatal("no")
	}

	b, err := json.Marshal(val)
	if err != nil {
		t.Fatal(err)
	}

	var out2 types.ThingWithSomeTime
	if err := json.Unmarshal(b, &out2); err != nil {
		t.Fatal(err)
	}

	if out2.When != out.When {
		t.Fatal(err)
	}

}

func TestLessToMoreFieldsRoundTrip(t *testing.T) {
	dummyCid, _ := cid.Parse("bafkqaaa")
	simpleTypeOne := types.SimpleTypeOne{
		Foo:     "foo",
		Value:   1,
		Binary:  []byte("bin"),
		Signed:  -1,
		NString: "namedstr",
		U8:      3,
		U16:     4,
		U32:     5,
		I8:      -3,
		I16:     -4,
		I32:     -5,
	}
	obj := &types.SimpleStructV1{
		OldStr:    "hello",
		OldBytes:  []byte("bytes"),
		OldNum:    10,
		OldPtr:    &dummyCid,
		OldMap:    map[string]types.SimpleTypeOne{"first": simpleTypeOne},
		OldArray:  []types.SimpleTypeOne{simpleTypeOne},
		OldStruct: simpleTypeOne,
	}

	buf := new(bytes.Buffer)
	if err := obj.MarshalCBOR(buf); err != nil {
		t.Fatal("failed marshaling", err)
	}

	enc := buf.Bytes()

	nobj := types.SimpleStructV2{}
	if err := nobj.UnmarshalCBOR(bytes.NewReader(enc)); err != nil {
		t.Logf("got bad bytes: %x", enc)
		t.Fatal("failed to round trip object: ", err)
	}

	if obj.OldStr != nobj.OldStr {
		t.Fatal("mismatch ", obj.OldStr, " != ", nobj.OldStr)
	}
	if nobj.NewStr != "" {
		t.Fatal("expected field to be zero value")
	}

	if obj.OldNum != nobj.OldNum {
		t.Fatal("mismatch ", obj.OldNum, " != ", nobj.OldNum)
	}
	if nobj.NewNum != 0 {
		t.Fatal("expected field to be zero value")
	}

	if !bytes.Equal(obj.OldBytes, nobj.OldBytes) {
		t.Fatal("mismatch ", obj.OldBytes, " != ", nobj.OldBytes)
	}
	if nobj.NewBytes != nil {
		t.Fatal("expected field to be zero value")
	}

	if *obj.OldPtr != *nobj.OldPtr {
		t.Fatal("mismatch ", obj.OldPtr, " != ", nobj.OldPtr)
	}
	if nobj.NewPtr != nil {
		t.Fatal("expected field to be zero value")
	}

	if !cmp.Equal(obj.OldMap, nobj.OldMap) {
		t.Fatal("mismatch map marshal / unmarshal")
	}
	if len(nobj.NewMap) != 0 {
		t.Fatal("expected field to be zero value")
	}

	if !cmp.Equal(obj.OldArray, nobj.OldArray) {
		t.Fatal("mismatch array marshal / unmarshal")
	}
	if len(nobj.NewArray) != 0 {
		t.Fatal("expected field to be zero value")
	}

	if !cmp.Equal(obj.OldStruct, nobj.OldStruct) {
		t.Fatal("mismatch struct marshal / unmarshal")
	}
	if !cmp.Equal(nobj.NewStruct, types.SimpleTypeOne{}) {
		t.Fatal("expected field to be zero value")
	}
}

func TestMoreToLessFieldsRoundTrip(t *testing.T) {
	dummyCid1, _ := cid.Parse("bafkqaaa")
	dummyCid2, _ := cid.Parse("bafkqaab")
	simpleType1 := types.SimpleTypeOne{
		Foo:     "foo",
		Value:   1,
		Binary:  []byte("bin"),
		Signed:  -1,
		NString: "namedstr",
		U8:      3,
		U16:     4,
		U32:     5,
		I8:      -3,
		I16:     -4,
		I32:     -5,
	}
	simpleType2 := types.SimpleTypeOne{
		Foo:     "bar",
		Value:   2,
		Binary:  []byte("bin2"),
		Signed:  -2,
		NString: "namedstr2",
		U8:      3,
		U16:     4,
		U32:     5,
		I8:      -3,
		I16:     -4,
		I32:     -5,
	}
	obj := &types.SimpleStructV2{
		OldStr:    "oldstr",
		NewStr:    "newstr",
		OldBytes:  []byte("oldbytes"),
		NewBytes:  []byte("newbytes"),
		OldNum:    10,
		NewNum:    11,
		OldPtr:    &dummyCid1,
		NewPtr:    &dummyCid2,
		OldMap:    map[string]types.SimpleTypeOne{"foo": simpleType1},
		NewMap:    map[string]types.SimpleTypeOne{"bar": simpleType2},
		OldArray:  []types.SimpleTypeOne{simpleType1},
		NewArray:  []types.SimpleTypeOne{simpleType1, simpleType2},
		OldStruct: simpleType1,
		NewStruct: simpleType2,
	}

	buf := new(bytes.Buffer)
	if err := obj.MarshalCBOR(buf); err != nil {
		t.Fatal("failed marshaling", err)
	}

	enc := buf.Bytes()

	nobj := types.SimpleStructV1{}
	if err := nobj.UnmarshalCBOR(bytes.NewReader(enc)); err != nil {
		t.Logf("got bad bytes: %x", enc)
		t.Fatal("failed to round trip object: ", err)
	}

	if obj.OldStr != nobj.OldStr {
		t.Fatal("mismatch", obj.OldStr, " != ", nobj.OldStr)
	}
	if obj.OldNum != nobj.OldNum {
		t.Fatal("mismatch ", obj.OldNum, " != ", nobj.OldNum)
	}
	if !bytes.Equal(obj.OldBytes, nobj.OldBytes) {
		t.Fatal("mismatch ", obj.OldBytes, " != ", nobj.OldBytes)
	}
	if *obj.OldPtr != *nobj.OldPtr {
		t.Fatal("mismatch ", obj.OldPtr, " != ", nobj.OldPtr)
	}
	if !cmp.Equal(obj.OldMap, nobj.OldMap) {
		t.Fatal("mismatch map marshal / unmarshal")
	}
	if !cmp.Equal(obj.OldArray, nobj.OldArray) {
		t.Fatal("mismatch array marshal / unmarshal")
	}
	if !cmp.Equal(obj.OldStruct, nobj.OldStruct) {
		t.Fatal("mismatch struct marshal / unmarshal")
	}
}
