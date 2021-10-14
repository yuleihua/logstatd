package cache

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasic(t *testing.T) {
	ts := NewCacheData("./noregtt")
	defer ts.Close()

	if err := ts.Put([]byte("test"), []byte("value")); err != nil {
		t.Fatalf("Put error")
	}

	ishas, _ := ts.Has([]byte("test"))
	assert.Equal(t, true, ishas)

	val, _ := ts.Get([]byte("test"))

	assert.Equal(t, string(val), "value")

	if err := ts.Delete([]byte("test")); err != nil {
		t.Fatalf("Delete error")
	}

	var e error
	val, e = ts.Get([]byte("test"))
	assert.Equal(t, string(val), "")

	fmt.Println(e)
}
