package lnroll

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/apg/ln"
	"github.com/heroku/x/scrub"
	"github.com/pkg/errors"
)

type Client interface {
	Critical(err error, extras map[string]string) (uuid string, e error)
	CriticalStack(err error, ptrs []uintptr, custom map[string]string) (uuid string, e error)
	Error(err error, extras map[string]string) (uuid string, e error)
	ErrorStack(err error, ptrs []uintptr, custom map[string]string) (uuid string, e error)
}

// grab a list of pointers to all of the functions in the callstack
type stackTracer interface {
	StackTrace() errors.StackTrace
}

func stripURLError(e error) error {
	switch e.(type) {
	case *url.Error:
		ue := e.(*url.Error)
		u, _ := url.Parse(ue.URL)
		ue.URL = scrub.URL(u).String()

		return ue
	}

	return e
}

// New returns a new FilterFunc which reports errors to Rollbar.
func New(client Client) ln.FilterFunc {
	return ln.FilterFunc(func(ctx context.Context, e ln.Event) bool {
		if e.Pri > ln.PriError {
			return true
		}

		// find the "err", or "error", and use that to report from F.
		var err error
		extras := make(map[string]string)

		for k, v := range e.Data {
			if err == nil && (k == "err" || k == "error") {
				if e, ok := v.(error); !ok {
					err = errors.New(toString(v))
				} else {
					err = stripURLError(e)
				}
			} else {
				extras[k] = toString(v)
			}
		}

		// e.Data was empty or e.Data["err"/"error"] didn't exist, so err is still nil.
		if err == nil {
			if e.Message != "" { // if we have a message though, may as well use that. This could happen via: ln.Error(fmt.Sprintf("ERROR!"))
				err = errors.New(e.Message)
			} else {
				if len(extras) == 0 { // nothing to report (no message, no extras)
					return true
				}
			}
		}

		defer func() {
			if r := recover(); r != nil {
				ln.Info(ctx, ln.F{"msg": "Panic while trying to report error to rollbar", "panic": true, "recover": true, "err": r})
			}
		}()

		if sterr, ok := err.(stackTracer); ok {
			// Have a stack, so let's prepare it.
			st := sterr.StackTrace()

			// Client requires a slice of uintptr, we have []errors.Frame, fix this
			pc := make([]uintptr, 0, len(st))
			for _, val := range st {
				pc = append(pc, uintptr(val))
			}

			// Select the function to use to report the error
			var rf func(error, []uintptr, map[string]string) (string, error)
			switch e.Pri {
			case ln.PriError:
				rf = client.ErrorStack
			case ln.PriCritical, ln.PriAlert, ln.PriEmergency:
				rf = client.CriticalStack
			}
			if uid, err := rf(err, pc, extras); err != nil {
				// These can't be Error or lnroll will recursively handle
				ln.Info(ctx, ln.F{"err": err, "uuid": uid, "priority": e.Pri.String(), "action": "rollbar-report"})
			}

			return true
		}

		var rf func(error, map[string]string) (string, error)
		switch e.Pri { // select the function to use to report the error
		case ln.PriError:
			rf = client.Error
		case ln.PriCritical, ln.PriAlert, ln.PriEmergency:
			rf = client.Critical
		}
		if uid, err := rf(err, extras); err != nil {
			// These can't be Error or lnroll will recursively handle
			ln.Info(ctx, ln.F{"err": err, "uuid": uid, "priority": e.Pri.String(), "action": "rollbar-report"})
		}
		return true

	})
}

func toString(v interface{}) string {
	switch t := v.(type) {
	case time.Time:
		return t.Format(time.RFC3339)
	default:
		if s, ok := v.(fmt.Stringer); ok {
			return s.String()
		}
		return fmt.Sprintf("%+v", v)
	}
}
