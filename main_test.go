package main

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestPatch_Patch(t *testing.T) {

	in := []byte("Hello, world patcher")
	old := hex.EncodeToString([]byte("Hello"))
	new := hex.EncodeToString([]byte("Bay"))
	out := []byte("Bay, world patcher")

	p := patch{
		Old: old,
		New: new,
	}

	outPatch, err := p.Patch(in)

	if err != nil {
		t.Error(err)
	}

	ok := bytes.Equal(out, outPatch)

	if !ok {
		t.Errorf("<%s> must be <%s>", string(outPatch), string(out))
	}

}
