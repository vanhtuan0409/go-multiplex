package multiplex

import (
	"bytes"
	"testing"
)

func TestEncoding(t *testing.T) {
	p := packet{
		StreamID: 1,
		Flag:     FFIN,
		Data:     []byte("Hello"),
	}

	buf := bytes.NewBuffer([]byte{})

	if err := encode(buf, p); err != nil {
		t.Errorf("Failed to encode data. ERR: %v", err)
	}

	out := packet{}
	if err := decode(buf, &out); err != nil {
		t.Errorf("Failed to decode data. ERR: %v", err)
	}
}

func TestFlag(t *testing.T) {
	p := packet{
		Flag: FSYNC | FACK,
	}

	if ok := p.Has(FSYNC); !ok {
		t.Error("Failed test 1")
	}

	if ok := p.Has(FACK); !ok {
		t.Error("Failed test 2")
	}

	if ok := p.Has(FSYNC, FACK); !ok {
		t.Error("Failed test 3")
	}

	if ok := p.Has(FFIN); ok {
		t.Error("Failed test 4")
	}
}
