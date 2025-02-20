package typegen

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"sort"
	"time"

	"github.com/ipfs/go-cid"
)

const maxCidLength = 100
const maxHeaderSize = 9

// discard is a helper function to discard data from a reader, special-casing
// the most common readers we encounter in this library for a significant
// performance boost.
func discard(br io.Reader, n int) error {
	switch r := br.(type) {
	case *bytes.Buffer:
		buf := r.Next(n)
		if len(buf) < n {
			return io.ErrUnexpectedEOF
		}
		return nil
	case *bytes.Reader:
		if r.Len() < n {
			_, _ = r.Seek(0, io.SeekEnd)
			return io.ErrUnexpectedEOF
		}
		_, err := r.Seek(int64(n), io.SeekCurrent)
		return err
	case *bufio.Reader:
		_, err := r.Discard(n)
		return err
	default:
		_, err := io.CopyN(ioutil.Discard, br, int64(n))
		return err
	}
}

func ScanForLinks(br io.Reader, cb func(cid.Cid)) (int, error) {
	bytesRead := 0

	scratch := make([]byte, maxCidLength)
	for remaining := uint64(1); remaining > 0; remaining-- {
		maj, extra, read, err := CborReadHeaderBuf(br, scratch)
		if err != nil {
			return bytesRead, err
		}
		bytesRead += read

		switch maj {
		case MajUnsignedInt, MajNegativeInt, MajOther:
		case MajByteString, MajTextString:
			err := discard(br, int(extra))
			if err != nil {
				return bytesRead, err
			}
			bytesRead += int(extra)
		case MajTag:
			if extra == 42 {
				maj, extra, read, err = CborReadHeaderBuf(br, scratch)
				if err != nil {
					return bytesRead, err
				}
				bytesRead += read

				if maj != MajByteString {
					return bytesRead, fmt.Errorf("expected cbor type 'byte string' in input")
				}

				if extra > maxCidLength {
					return bytesRead, fmt.Errorf("string in cbor input too long")
				}

				if read, err := io.ReadAtLeast(br, scratch[:extra], int(extra)); err != nil {
					return bytesRead, err
				} else {
					bytesRead += read
				}

				c, err := cid.Cast(scratch[1:extra])
				if err != nil {
					return bytesRead, err
				}
				cb(c)

			} else {
				remaining++
			}
		case MajArray:
			remaining += extra
		case MajMap:
			remaining += (extra * 2)
		default:
			return bytesRead, fmt.Errorf("unhandled cbor type: %d", maj)
		}
	}
	return bytesRead, nil
}

const (
	MajUnsignedInt = 0
	MajNegativeInt = 1
	MajByteString  = 2
	MajTextString  = 3
	MajArray       = 4
	MajMap         = 5
	MajTag         = 6
	MajOther       = 7
)

var maxLengthError = fmt.Errorf("length beyond maximum allowed")

type Initter interface {
	// InitNilEmbeddedStruct prepare the struct for marshalling & unmarshalling by
	// recursively initialize the structs embedded by pointer to their zero values.
	InitNilEmbeddedStruct()
}

type CBORUnmarshaler interface {
	// UnmarshalCBOR reads CBOR bytes from io.Reader and unmarshals them into the Go type,
	// it returns the length of the unmarshaled bytes and the error if occurred.
	UnmarshalCBOR(io.Reader) (int, error)
}

type CBORMarshaler interface {
	// MarshalCBOR marshals the Go type into CBOR bytes and writes them into io.Writer,
	// it returns the length of the marshaled bytes written and the error if occurred.
	MarshalCBOR(io.Writer) (int, error)
}

type Deferred struct {
	Raw []byte
}

func (d *Deferred) MarshalCBOR(w io.Writer) (int, error) {
	if d == nil {
		return w.Write(CborNull)
	}
	if d.Raw == nil {
		return 0, errors.New("cannot marshal Deferred with nil value for Raw (will not unmarshal)")
	}
	return w.Write(d.Raw)
}

func (d *Deferred) UnmarshalCBOR(br io.Reader) (int, error) {
	bytesRead := 0

	// Reuse any existing buffers.
	reusedBuf := d.Raw[:0]
	d.Raw = nil
	buf := bytes.NewBuffer(reusedBuf)

	// Allocate some scratch space.
	scratch := make([]byte, maxHeaderSize)

	// Algorithm:
	//
	// 1. We start off expecting to read one element.
	// 2. If we see a tag, we expect to read one more element so we increment "remaining".
	// 3. If see an array, we expect to read "extra" elements so we add "extra" to "remaining".
	// 4. If see a map, we expect to read "2*extra" elements so we add "2*extra" to "remaining".
	// 5. While "remaining" is non-zero, read more elements.

	// define this once so we don't keep allocating it.
	limitedReader := io.LimitedReader{R: br}
	for remaining := uint64(1); remaining > 0; remaining-- {
		maj, extra, read, err := CborReadHeaderBuf(br, scratch)
		if err != nil {
			return bytesRead, err
		}
		bytesRead += read

		if _, err := WriteMajorTypeHeaderBuf(scratch, buf, maj, extra); err != nil {
			return bytesRead, err
		}

		switch maj {
		case MajUnsignedInt, MajNegativeInt, MajOther:
			// nothing fancy to do
		case MajByteString, MajTextString:
			if extra > ByteArrayMaxLen {
				return bytesRead, maxLengthError
			}
			// Copy the bytes
			limitedReader.N = int64(extra)
			buf.Grow(int(extra))
			if n, err := buf.ReadFrom(&limitedReader); err != nil {
				return bytesRead, err
			} else {
				bytesRead += int(n)
				if n < int64(extra) {
					return bytesRead, io.ErrUnexpectedEOF
				}
			}
		case MajTag:
			remaining++
		case MajArray:
			if extra > MaxLength {
				return bytesRead, maxLengthError
			}
			remaining += extra
		case MajMap:
			if extra > MaxLength {
				return bytesRead, maxLengthError
			}
			remaining += extra * 2
		default:
			return bytesRead, fmt.Errorf("unhandled deferred cbor type: %d", maj)
		}
	}
	d.Raw = buf.Bytes()
	return bytesRead, nil
}

func readByte(r io.Reader) (byte, error) {
	// try to cast to a concrete type, it's much faster than casting to an
	// interface.
	switch r := r.(type) {
	case *bytes.Buffer:
		return r.ReadByte()
	case *bytes.Reader:
		return r.ReadByte()
	case *bufio.Reader:
		return r.ReadByte()
	case *peeker:
		return r.ReadByte()
	case io.ByteReader:
		return r.ReadByte()
	}
	var buf [1]byte
	_, err := io.ReadFull(r, buf[:1])
	return buf[0], err
}

func CborReadHeader(br io.Reader) (byte, uint64, int, error) {
	bytesRead := 0

	first, err := readByte(br)
	if err != nil {
		return 0, 0, bytesRead, err
	}
	bytesRead++

	maj := (first & 0xe0) >> 5
	low := first & 0x1f

	switch {
	case low < 24:
		return maj, uint64(low), bytesRead, nil
	case low == 24:
		next, err := readByte(br)
		if err != nil {
			return 0, 0, bytesRead, err
		}
		bytesRead++

		if next < 24 {
			return 0, 0, bytesRead, fmt.Errorf("cbor input was not canonical (lval 24 with value < 24)")
		}
		return maj, uint64(next), bytesRead, nil
	case low == 25:
		scratch := make([]byte, 2)
		if read, err := io.ReadAtLeast(br, scratch[:2], 2); err != nil {
			return 0, 0, bytesRead, err
		} else {
			bytesRead += read
		}
		val := uint64(binary.BigEndian.Uint16(scratch[:2]))
		if val <= math.MaxUint8 {
			return 0, 0, bytesRead, fmt.Errorf("cbor input was not canonical (lval 25 with value <= MaxUint8)")
		}
		return maj, val, bytesRead, nil
	case low == 26:
		scratch := make([]byte, 4)
		if read, err := io.ReadAtLeast(br, scratch[:4], 4); err != nil {
			return 0, 0, bytesRead, err
		} else {
			bytesRead += read
		}
		val := uint64(binary.BigEndian.Uint32(scratch[:4]))
		if val <= math.MaxUint16 {
			return 0, 0, bytesRead, fmt.Errorf("cbor input was not canonical (lval 26 with value <= MaxUint16)")
		}
		return maj, val, bytesRead, nil
	case low == 27:
		scratch := make([]byte, 8)
		if read, err := io.ReadAtLeast(br, scratch, 8); err != nil {
			return 0, 0, bytesRead, err
		} else {
			bytesRead += read
		}
		val := binary.BigEndian.Uint64(scratch)
		if val <= math.MaxUint32 {
			return 0, 0, bytesRead, fmt.Errorf("cbor input was not canonical (lval 27 with value <= MaxUint32)")
		}
		return maj, val, bytesRead, nil
	default:
		return 0, 0, bytesRead, fmt.Errorf("invalid header: (%x)", first)
	}
}

func readByteBuf(r io.Reader, scratch []byte) (byte, error) {
	// Reading a single byte from these buffers is much faster than copying
	// into a slice.
	switch r := r.(type) {
	case *bytes.Buffer:
		return r.ReadByte()
	case *bytes.Reader:
		return r.ReadByte()
	case *bufio.Reader:
		return r.ReadByte()
	}
	n, err := r.Read(scratch[:1])
	if err != nil {
		return 0, err
	}
	if n != 1 {
		return 0, fmt.Errorf("failed to read a byte")
	}
	return scratch[0], err
}

// same as the above, just tries to allocate less by using a passed in scratch buffer
func CborReadHeaderBuf(br io.Reader, scratch []byte) (byte, uint64, int, error) {
	bytesRead := 0

	first, err := readByteBuf(br, scratch)
	if err != nil {
		return 0, 0, bytesRead, err
	}
	bytesRead++

	maj := (first & 0xe0) >> 5
	low := first & 0x1f

	switch {
	case low < 24:
		return maj, uint64(low), bytesRead, nil
	case low == 24:
		next, err := readByteBuf(br, scratch)
		if err != nil {
			return 0, 0, bytesRead, err
		}
		bytesRead++
		if next < 24 {
			return 0, 0, bytesRead, fmt.Errorf("cbor input was not canonical (lval 24 with value < 24)")
		}
		return maj, uint64(next), bytesRead, nil
	case low == 25:
		if read, err := io.ReadAtLeast(br, scratch[:2], 2); err != nil {
			return 0, 0, bytesRead, err
		} else {
			bytesRead += read
		}
		val := uint64(binary.BigEndian.Uint16(scratch[:2]))
		if val <= math.MaxUint8 {
			return 0, 0, bytesRead, fmt.Errorf("cbor input was not canonical (lval 25 with value <= MaxUint8)")
		}
		return maj, val, bytesRead, nil
	case low == 26:
		if read, err := io.ReadAtLeast(br, scratch[:4], 4); err != nil {
			return 0, 0, bytesRead, err
		} else {
			bytesRead += read
		}
		val := uint64(binary.BigEndian.Uint32(scratch[:4]))
		if val <= math.MaxUint16 {
			return 0, 0, bytesRead, fmt.Errorf("cbor input was not canonical (lval 26 with value <= MaxUint16)")
		}
		return maj, val, bytesRead, nil
	case low == 27:
		if read, err := io.ReadAtLeast(br, scratch[:8], 8); err != nil {
			return 0, 0, bytesRead, err
		} else {
			bytesRead += read
		}
		val := binary.BigEndian.Uint64(scratch[:8])
		if val <= math.MaxUint32 {
			return 0, 0, bytesRead, fmt.Errorf("cbor input was not canonical (lval 27 with value <= MaxUint32)")
		}
		return maj, val, bytesRead, nil
	default:
		return 0, 0, bytesRead, fmt.Errorf("invalid header: (%x)", first)
	}
}

func CborWriteHeader(w io.Writer, t byte, l uint64) (n int, err error) {
	return WriteMajorTypeHeader(w, t, l)
}

// TODO: No matter what I do, this function *still* allocates. Its super frustrating.
// See issue: https://github.com/golang/go/issues/33160
func WriteMajorTypeHeader(w io.Writer, t byte, l uint64) (n int, err error) {
	switch {
	case l < 24:
		return w.Write([]byte{(t << 5) | byte(l)})
	case l < (1 << 8):
		return w.Write([]byte{(t << 5) | 24, byte(l)})
	case l < (1 << 16):
		var b [3]byte
		b[0] = (t << 5) | 25
		binary.BigEndian.PutUint16(b[1:3], uint16(l))
		return w.Write(b[:])
	case l < (1 << 32):
		var b [5]byte
		b[0] = (t << 5) | 26
		binary.BigEndian.PutUint32(b[1:5], uint32(l))
		return w.Write(b[:])
	default:
		var b [9]byte
		b[0] = (t << 5) | 27
		binary.BigEndian.PutUint64(b[1:], uint64(l))
		return w.Write(b[:])
	}
}

// Same as the above, but uses a passed in buffer to avoid allocations
func WriteMajorTypeHeaderBuf(buf []byte, w io.Writer, t byte, l uint64) (n int, err error) {
	switch {
	case l < 24:
		buf[0] = (t << 5) | byte(l)
		return w.Write(buf[:1])
	case l < (1 << 8):
		buf[0] = (t << 5) | 24
		buf[1] = byte(l)
		return w.Write(buf[:2])
	case l < (1 << 16):
		buf[0] = (t << 5) | 25
		binary.BigEndian.PutUint16(buf[1:3], uint16(l))
		return w.Write(buf[:3])
	case l < (1 << 32):
		buf[0] = (t << 5) | 26
		binary.BigEndian.PutUint32(buf[1:5], uint32(l))
		return w.Write(buf[:5])
	default:
		buf[0] = (t << 5) | 27
		binary.BigEndian.PutUint64(buf[1:9], uint64(l))
		return w.Write(buf[:9])
	}
}

func CborEncodeMajorType(t byte, l uint64) []byte {
	switch {
	case l < 24:
		var b [1]byte
		b[0] = (t << 5) | byte(l)
		return b[:1]
	case l < (1 << 8):
		var b [2]byte
		b[0] = (t << 5) | 24
		b[1] = byte(l)
		return b[:2]
	case l < (1 << 16):
		var b [3]byte
		b[0] = (t << 5) | 25
		binary.BigEndian.PutUint16(b[1:3], uint16(l))
		return b[:3]
	case l < (1 << 32):
		var b [5]byte
		b[0] = (t << 5) | 26
		binary.BigEndian.PutUint32(b[1:5], uint32(l))
		return b[:5]
	default:
		var b [9]byte
		b[0] = (t << 5) | 27
		binary.BigEndian.PutUint64(b[1:], uint64(l))
		return b[:]
	}
}

func ReadTaggedByteArray(br io.Reader, exptag uint64, maxlen uint64) ([]byte, int, error) {
	bytesRead := 0

	maj, extra, read, err := CborReadHeader(br)
	if err != nil {
		return nil, bytesRead, err
	}
	bytesRead += read

	if maj != MajTag {
		return nil, bytesRead, fmt.Errorf("expected cbor type 'tag' in input")
	}

	if extra != exptag {
		return nil, bytesRead, fmt.Errorf("expected tag %d", exptag)
	}

	bs, read, err := ReadByteArray(br, maxlen)
	if err == nil {
		bytesRead += read
	}

	return bs, bytesRead, err
}

func ReadByteArray(br io.Reader, maxlen uint64) ([]byte, int, error) {
	bytesRead := 0

	maj, extra, read, err := CborReadHeader(br)
	if err != nil {
		return nil, bytesRead, err
	}
	bytesRead += read

	if maj != MajByteString {
		return nil, bytesRead, fmt.Errorf("expected cbor type 'byte string' in input")
	}

	if extra > maxlen {
		return nil, bytesRead, fmt.Errorf("string in cbor input too long, maxlen: %d", maxlen)
	}

	buf := make([]byte, extra)
	if read, err := io.ReadAtLeast(br, buf, int(extra)); err != nil {
		return nil, bytesRead, err
	} else {
		bytesRead += read
	}

	return buf, bytesRead, nil
}

var (
	CborBoolFalse = []byte{0xf4}
	CborBoolTrue  = []byte{0xf5}
	CborNull      = []byte{0xf6}
)

func EncodeBool(b bool) []byte {
	if b {
		return CborBoolTrue
	}
	return CborBoolFalse
}

func WriteBool(w io.Writer, b bool) (n int, err error) {
	return w.Write(EncodeBool(b))
}

func ReadString(r io.Reader) (string, int, error) {
	bytesRead := 0

	maj, l, read, err := CborReadHeader(r)
	if err != nil {
		return "", bytesRead, err
	}
	bytesRead += read

	if maj != MajTextString {
		return "", bytesRead, fmt.Errorf("got tag %d while reading string value (l = %d)", maj, l)
	}

	if l > MaxLength {
		return "", bytesRead, fmt.Errorf("string in input was too long")
	}

	buf := make([]byte, l)
	read, err = io.ReadAtLeast(r, buf, int(l))
	if err != nil {
		return "", bytesRead, err
	}
	bytesRead += read

	return string(buf), bytesRead, nil
}

func ReadStringBuf(r io.Reader, scratch []byte) (string, int, error) {
	bytesRead := 0

	maj, l, read, err := CborReadHeaderBuf(r, scratch)
	if err != nil {
		return "", bytesRead, err
	}
	bytesRead += read

	if maj != MajTextString {
		return "", bytesRead, fmt.Errorf("got tag %d while reading string value (l = %d)", maj, l)
	}

	if l > MaxLength {
		return "", bytesRead, fmt.Errorf("string in input was too long")
	}

	buf := make([]byte, l)
	read, err = io.ReadAtLeast(r, buf, int(l))
	if err != nil {
		return "", bytesRead, err
	}
	bytesRead += read

	return string(buf), bytesRead, nil
}

func ReadCid(br io.Reader) (cid.Cid, int, error) {
	bytesRead := 0

	buf, read, err := ReadTaggedByteArray(br, 42, 512)
	if err != nil {
		return cid.Undef, bytesRead, err
	}
	bytesRead += read

	cid, err := bufToCid(buf)
	return cid, bytesRead, err
}

func bufToCid(buf []byte) (cid.Cid, error) {

	if len(buf) == 0 {
		return cid.Undef, fmt.Errorf("undefined cid")
	}

	if len(buf) < 2 {
		return cid.Undef, fmt.Errorf("cbor serialized CIDs must have at least two bytes")
	}

	if buf[0] != 0 {
		return cid.Undef, fmt.Errorf("cbor serialized CIDs must have binary multibase")
	}

	return cid.Cast(buf[1:])
}

var byteArrZero = []byte{0}

func WriteCid(w io.Writer, c cid.Cid) (n int, err error) {
	if n_, err := WriteMajorTypeHeader(w, MajTag, 42); err != nil {
		return n + n_, err
	} else {
		n += n_
	}
	if c == cid.Undef {
		return n, fmt.Errorf("undefined cid")
		//return CborWriteHeader(w, MajByteString, 0)
	}

	if n_, err := WriteMajorTypeHeader(w, MajByteString, uint64(c.ByteLen()+1)); err != nil {
		return n + n_, err
	} else {
		n += n_
	}

	// that binary multibase prefix...
	if n_, err := w.Write(byteArrZero); err != nil {
		return n + n_, err
	} else {
		n += n_
	}

	if n_, err := c.WriteBytes(w); err != nil {
		return n + n_, err
	} else {
		n += n_
	}

	return n, nil
}

func WriteCidBuf(buf []byte, w io.Writer, c cid.Cid) (n int, er error) {
	if n_, err := WriteMajorTypeHeaderBuf(buf, w, MajTag, 42); err != nil {
		return n + n_, err
	} else {
		n += n_
	}
	if c == cid.Undef {
		return n, fmt.Errorf("undefined cid")
		//return CborWriteHeader(w, MajByteString, 0)
	}

	if n_, err := WriteMajorTypeHeaderBuf(buf, w, MajByteString, uint64(c.ByteLen()+1)); err != nil {
		return n + n_, err
	} else {
		n += n_
	}

	// that binary multibase prefix...
	if n_, err := w.Write(byteArrZero); err != nil {
		return n + n_, err
	} else {
		n += n_
	}

	if n_, err := c.WriteBytes(w); err != nil {
		return n + n_, err
	} else {
		n += n_
	}

	return n, nil
}

type CborBool bool

func (cb CborBool) MarshalCBOR(w io.Writer) (n int, err error) {
	return WriteBool(w, bool(cb))
}

func (cb *CborBool) UnmarshalCBOR(r io.Reader) (int, error) {
	bytesRead := 0

	t, val, read, err := CborReadHeader(r)
	if err != nil {
		return bytesRead, err
	}
	bytesRead += read

	if t != MajOther {
		return bytesRead, fmt.Errorf("booleans should be major type 7")
	}

	switch val {
	case 20:
		*cb = false
	case 21:
		*cb = true
	default:
		return bytesRead, fmt.Errorf("booleans are either major type 7, value 20 or 21 (got %d)", val)
	}
	return bytesRead, nil
}

type CborInt int64

func (ci CborInt) MarshalCBOR(w io.Writer) (n int, err error) {
	v := int64(ci)
	if v >= 0 {
		if n_, err := WriteMajorTypeHeader(w, MajUnsignedInt, uint64(v)); err != nil {
			return n + n_, err
		} else {
			n += n_
		}
	} else {
		if n_, err := WriteMajorTypeHeader(w, MajNegativeInt, uint64(-v)-1); err != nil {
			return n + n_, err
		} else {
			n += n_
		}
	}
	return n, nil
}

func (ci *CborInt) UnmarshalCBOR(r io.Reader) (int, error) {
	bytesRead := 0

	maj, extra, read, err := CborReadHeader(r)
	if err != nil {
		return bytesRead, err
	}
	bytesRead += read

	var extraI int64
	switch maj {
	case MajUnsignedInt:
		extraI = int64(extra)
		if extraI < 0 {
			return bytesRead, fmt.Errorf("int64 positive overflow")
		}
	case MajNegativeInt:
		extraI = int64(extra)
		if extraI < 0 {
			return bytesRead, fmt.Errorf("int64 negative oveflow")
		}
		extraI = -1 - extraI
	default:
		return bytesRead, fmt.Errorf("wrong type for int64 field: %d", maj)
	}

	*ci = CborInt(extraI)
	return bytesRead, nil
}

type CborTime time.Time

func (ct CborTime) MarshalCBOR(w io.Writer) (n int, err error) {
	nsecs := ct.Time().UnixNano()

	cbi := CborInt(nsecs)

	return cbi.MarshalCBOR(w)
}

func (ct *CborTime) UnmarshalCBOR(r io.Reader) (int, error) {
	bytesRead := 0

	var cbi CborInt
	if read, err := cbi.UnmarshalCBOR(r); err != nil {
		return bytesRead, err
	} else {
		bytesRead += read
	}

	t := time.Unix(0, int64(cbi))

	*ct = (CborTime)(t)
	return bytesRead, nil
}

func (ct CborTime) Time() time.Time {
	return (time.Time)(ct)
}

func (ct CborTime) MarshalJSON() ([]byte, error) {
	return ct.Time().MarshalJSON()
}

func (ct *CborTime) UnmarshalJSON(b []byte) error {
	var t time.Time
	if err := t.UnmarshalJSON(b); err != nil {
		return err
	}
	*(*time.Time)(ct) = t
	return nil
}

func MapKeySort_RFC7049(keys []string) {
	sort.Slice(keys, func(i, j int) bool {
		return mapKeySort_RFC7049Less(keys[i], keys[j])
	})
}

func mapKeySort_RFC7049Less(k1 string, k2 string) bool {
	li, lj := len(k1), len(k2)
	if li == lj {
		return k1 < k2
	}
	return li < lj
}
