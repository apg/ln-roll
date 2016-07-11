package lnroll

import (
	"testing"

	"github.com/apg/ln"
)

type mockClient struct {
	C int
	E int
}

func (c *mockClient) Critical(err error, extras map[string]string) (uuid string, e error) {
	c.C++
	return
}

func (c *mockClient) Error(err error, extras map[string]string) (uuid string, e error) {
	c.E++
	return
}

func TestFilter(t *testing.T) {
	m := &mockClient{}
	underTest := New(m)

	log := &ln.Logger{
		Pri:     ln.PriError,
		Filters: []ln.Filter{underTest},
	}

	log.Error(ln.F{"err": "Here's an error"})

	if m.E == 0 {
		t.Fatalf("Filter didn't fire on Error")
	}
}
