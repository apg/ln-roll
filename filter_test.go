package lnroll

import (
	"testing"

	"github.com/apg/ln"
)

type Client interface {
	Critical(err error, extras map[string]string) (uuid string, e error)
	Error(err error, extras map[string]string) (uuid string, e error)
	Warning(err error, extras map[string]string) (uuid string, e error)
	Info(msg string, extras map[string]string) (uuid string, e error)
	Debug(msg string, extras map[string]string) (uuid string, e error)
}

type mockClient struct {
	C int
	E int
	W int
	I int
	D int
}

func (c *mockClient) Critical(err error, extras map[string]string) (uuid string, e error) {
	c.C++
	return
}

func (c *mockClient) Error(err error, extras map[string]string) (uuid string, e error) {
	c.E++
	return
}

func (c *mockClient) Warning(err error, extras map[string]string) (uuid string, e error) {
	c.W++
	return
}

func (c *mockClient) Info(thing string, extras map[string]string) (uuid string, e error) {
	c.I++
	return
}

func (c *mockClient) Debug(thing string, extras map[string]string) (uuid string, e error) {
	c.D++
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

	if m.E != 0 {
		t.Fatalf("Filter didn't fire on Error")
	}
}
