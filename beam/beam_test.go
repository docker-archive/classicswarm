package beam

import (
	"testing"
)

func TestModes(t *testing.T) {
	if Ret == 0 {
		t.Fatalf("0")
	}
}

func TestRetPipe(t *testing.T) {
	var (
		shouldBeEqual = RetPipe
	)
	if RetPipe != shouldBeEqual {
		t.Fatalf("%#v should equal %#v", RetPipe, shouldBeEqual)
	}
}
