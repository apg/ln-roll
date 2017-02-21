package lnroll

import (
	"fmt"
	"testing"

	"github.com/apg/ln"
	"github.com/pkg/errors"
)

type mockClient struct {
	C  int
	E  int
	CS int
	ES int
}

func (c *mockClient) Critical(err error, extras map[string]string) (uuid string, e error) {
	c.C++
	return
}

func (c *mockClient) CriticalStack(err error, pc []uintptr, extras map[string]string) (uuid string, e error) {
	if len(pc) > 0 {
		c.CS++
	}
	return
}

func (c *mockClient) Error(err error, extras map[string]string) (uuid string, e error) {
	c.E++
	return
}

func (c *mockClient) ErrorStack(err error, pc []uintptr, extras map[string]string) (uuid string, e error) {
	if len(pc) > 0 {
		c.ES++
	}
	return
}

func TestFilter(t *testing.T) {
	m := &mockClient{}
	underTest := New(m)

	log := &ln.Logger{
		Pri:     ln.PriInfo,
		Filters: []ln.Filter{underTest},
	}

	log.Error(ln.F{"err": fmt.Errorf("an error without a stack")})
	if m.E == 0 {
		t.Errorf("Filter didn't fire on Error %+v", m)
	}

	log.Critical(ln.F{"err": fmt.Errorf("an error without a stack")})
	if m.C == 0 {
		t.Errorf("Filter didn't fire on Critical %+v", m)
	}
}

func TestFilterWithStack(t *testing.T) {
	m := &mockClient{}
	underTest := New(m)

	log := &ln.Logger{
		Pri:     ln.PriInfo,
		Filters: []ln.Filter{underTest},
	}

	err := errors.Wrap(errors.New("hi"), "stack")
	log.Error(ln.F{"err": err})
	if m.ES == 0 {
		t.Errorf("Filter didn't fire on Error %+v", *m)
	}

	err = errors.Wrap(errors.New("hi"), "stack 2")
	log.Critical(ln.F{"err": err})
	if m.CS == 0 {
		t.Errorf("Filter didn't fire on Critical %+v", *m)
	}
}
