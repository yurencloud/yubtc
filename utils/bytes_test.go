package utils

import "testing"

func TestBytesCombine(t *testing.T) {
	r := BytesCombine([]byte("hello"), []byte("world"))
	r2 := BytesCombine([]byte("world"), []byte("hello"))
	t.Log(r)
	t.Log(r2)
}
