package filedrive

import (
	"bytes"
	"io"
	"testing"

	"github.com/moov-io/ach"
)

func TestACHReaderWithTeeReader(t *testing.T) {
	input := []byte("ftp-test")

	var buf bytes.Buffer
	var r io.Reader = bytes.NewReader(input)

	tee := io.TeeReader(r, &buf)

	reader := ach.NewReader(tee)

	_, err := reader.Read()

	t.Logf("input: %q", input)
	t.Logf("buffer: %q", buf.String())
	t.Logf("error: %v", err)

	if buf.String() != string(input) {
		t.Fatalf("expected buffer %q, got %q", input, buf.String())
	}

	if err == nil {
		t.Fatal("expected invalid ACH file to return an error")
	}
}
