package lnroll

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/apg/ln"
	"github.com/pkg/errors"
)

var ctx context.Context

func init() { ctx = context.Background() }

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
	if err == nil {
		panic("err is nil")
	}
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

	log.Error(ctx, ln.F{"err": fmt.Errorf("an error without a stack")})
	if m.E == 0 {
		t.Errorf("Filter didn't fire on Error %+v", m)
	}

	log.Critical(ctx, ln.F{"err": fmt.Errorf("an error without a stack")})
	if m.C == 0 {
		t.Errorf("Filter didn't fire on Critical %+v", m)
	}
}

func TestFilterWithString(t *testing.T) {
	var m mockClient
	underTest := New(&m)

	log := ln.Logger{
		Pri:     ln.PriError,
		Filters: []ln.Filter{underTest},
	}

	log.Error(ctx, fmt.Sprintf("ERROR!"))
	log.Error(ctx, nil)
	log.Error(ctx)
	var xs []interface{}
	log.Error(ctx, xs)
}

func TestFilterWithStack(t *testing.T) {
	m := &mockClient{}
	underTest := New(m)

	log := &ln.Logger{
		Pri:     ln.PriInfo,
		Filters: []ln.Filter{underTest},
	}

	err := errors.Wrap(errors.New("hi"), "stack")
	log.Error(ctx, ln.F{"err": err})
	if m.ES == 0 {
		t.Errorf("Filter didn't fire on Error %+v", *m)
	}

	err = errors.Wrap(errors.New("hi"), "stack 2")
	log.Critical(ctx, ln.F{"err": err})
	if m.CS == 0 {
		t.Errorf("Filter didn't fire on Critical %+v", *m)
	}
}

func TestStripURLError(t *testing.T) {
	cases := []struct {
		name         string
		err          error
		secretToFind string
	}{
		{
			name: "passthru",
			err:  errors.New("test"),
		},
		{
			name: "scrub secrets",
			err: &url.Error{
				Op:  "pan gangnam style",
				URL: "http://AzureDiamond:hunter2@127.0.0.1/",
				Err: errors.New("test"),
			},
		},
	}

	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			res := stripURLError(cs.err)
			if cs.secretToFind != "" {
				es := res.Error()
				if strings.Contains(es, cs.secretToFind) {
					t.Fatal("stripURLError didn't strip the password: %v", es)
				}
			}
		})
	}
}
