package m3u

import (
	"fmt"
	"io"
)

type Marshaler interface {
	MarshalM3U() ([]byte, error)
}

type Unmarshaler interface {
	UnmarshalM3U([]byte) error
}

// Unmarshal parses the M3U-encoded data and stores the result in the value pointed to by v.
func Unmarshal(data []byte, v Unmarshaler) error {
	return v.UnmarshalM3U(data)
}

func Marshal(v Marshaler) ([]byte, error) {
	return v.MarshalM3U()
}

type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func (e *Encoder) Encode(v Marshaler) error {
	playlist, ok := v.(*Playlist)
	if !ok {
		return fmt.Errorf("unsupported type: %T", v)
	}
	return playlist.marshalM3U(e.w)
}
