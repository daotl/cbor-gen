package testing_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/ipfs/go-cid"

	cbg "github.com/daotl/cbor-gen"
	types "github.com/daotl/cbor-gen/testing"
)

func BenchmarkMarshaling(b *testing.B) {
	r := rand.New(rand.NewSource(56887))
	val, ok := quick.Value(reflect.TypeOf(types.SimpleTypeTwo{}), r)
	if !ok {
		b.Fatal("failed to construct type")
	}

	tt := val.Interface().(types.SimpleTypeTwo)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := tt.MarshalCBOR(ioutil.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshaling(b *testing.B) {
	r := rand.New(rand.NewSource(123456))
	val, ok := quick.Value(reflect.TypeOf(types.SimpleTypeTwo{}), r)
	if !ok {
		b.Fatal("failed to construct type")
	}

	tt := val.Interface().(types.SimpleTypeTwo)

	buf := new(bytes.Buffer)
	if err := tt.MarshalCBOR(buf); err != nil {
		b.Fatal(err)
	}

	reader := bytes.NewReader(buf.Bytes())

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader.Seek(0, io.SeekStart)
		var tt types.SimpleTypeTwo
		if err := tt.UnmarshalCBOR(reader); err != nil {
			b.Fatal(err)
		}
	}

}

func BenchmarkLinkScan(b *testing.B) {
	r := rand.New(rand.NewSource(123456))
	val, ok := quick.Value(reflect.TypeOf(types.SimpleTypeTwo{}), r)
	if !ok {
		b.Fatal("failed to construct type")
	}

	tt := val.Interface().(types.SimpleTypeTwo)

	buf := new(bytes.Buffer)
	if err := tt.MarshalCBOR(buf); err != nil {
		b.Fatal(err)
	}

	reader := bytes.NewReader(buf.Bytes())

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader.Seek(0, io.SeekStart)
		if err := cbg.ScanForLinks(reader, func(cid.Cid) {}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDeferred(b *testing.B) {
	r := rand.New(rand.NewSource(123456))
	val, ok := quick.Value(reflect.TypeOf(types.SimpleTypeTwo{}), r)
	if !ok {
		b.Fatal("failed to construct type")
	}

	tt := val.Interface().(types.SimpleTypeTwo)

	buf := new(bytes.Buffer)
	if err := tt.MarshalCBOR(buf); err != nil {
		b.Fatal(err)
	}

	var (
		deferred cbg.Deferred
		reader   = bytes.NewReader(buf.Bytes())
	)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader.Seek(0, io.SeekStart)
		if err := deferred.UnmarshalCBOR(reader); err != nil {
			b.Fatal(err)
		}
	}
}
