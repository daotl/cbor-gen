package typegen

import (
	"bytes"
	"testing"
)

func TestValidateShort(t *testing.T) {
	var buf bytes.Buffer
	if n, err := WriteMajorTypeHeader(&buf, MajByteString, 100); err != nil {
		t.Fatal("failed to write header")
	} else if n != buf.Len() {
		t.Fatal("returned length does not match the byte length")
	}
	if err := ValidateCBOR(buf.Bytes()); err == nil {
		t.Fatal("expected an error checking truncated cbor")
	}
}

func TestValidateDouble(t *testing.T) {
	var buf bytes.Buffer
	n := 0
	if n_, err := WriteBool(&buf, false); err != nil {
		t.Fatal(err)
	} else if n+n_ != buf.Len() {
		t.Fatal("returned length does not match the byte length")
	} else {
		n += n_
	}
	if n_, err := WriteBool(&buf, false); err != nil {
		t.Fatal(err)
	} else if n+n_ != buf.Len() {
		t.Fatal("returned length does not match the byte length")
	} else {
		n += n
	}

	if err := ValidateCBOR(buf.Bytes()); err == nil {
		t.Fatal("expected an error checking cbor with two objects")
	}
}

func TestValidate(t *testing.T) {
	var buf bytes.Buffer
	if n, err := WriteBool(&buf, false); err != nil {
		t.Fatal(err)
	} else if n != buf.Len() {
		t.Fatal("returned length does not match the byte length")
	}

	if err := ValidateCBOR(buf.Bytes()); err != nil {
		t.Fatal(err)
	}
}
