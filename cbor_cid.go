package typegen

import (
	"io"

	"github.com/ipfs/go-cid"
)

type CborCid cid.Cid

func (c CborCid) MarshalCBOR(w io.Writer) error {
	return WriteCid(w, cid.Cid(c))
}

func (c *CborCid) UnmarshalCBOR(r io.Reader) (int, error) {
	oc, read, err := ReadCid(r)
	if err != nil {
		return 0, err
	}
	*c = CborCid(oc)
	return read, nil
}
