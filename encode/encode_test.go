package encode

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestEncodeZx0(t *testing.T) {
	fr, err := os.Open("../1.scr")
	if err != nil {
		t.Fatalf("%s\n", err)
	}
	buf, err := ioutil.ReadAll(fr)
	if err != nil {
		t.Fatalf("%s\n", err)
	}
	out := Encode(buf)
	if len(out) != 20486 {
		t.Fatalf("Expected size 20486 and gets %d\n", len(out))
	}
}
