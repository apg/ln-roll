package lnroll

import (
	"fmt"
	"time"

	"github.com/apg/ln"
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

// New returns a new FilterFunc which reports errors to Rollbar.
func New(client Client) ln.FilterFunc {
	return ln.FilterFunc(func(e ln.Event) bool {
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
					err = e
				}
			} else {
				extras[k] = toString(v)
			}
		}

		// e.Data was empty or e.Data["err"/"error"] didn't exist, so err wasn't set.
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
				ln.Info(ln.F{"msg": "Panic while trying to report error to rollbar", "panic": true, "recover": true, "err": r})
			}
		}()

		sterr, ok := err.(stackTracer)
		if !ok {
			switch e.Pri {
			case ln.PriError:
				uid, err := client.Error(err, extras)
				if err != nil {
					// These can't be Error or lnroll will recursively handle
					ln.Info(ln.F{"err": err, "uuid": uid, "priority": e.Pri.String(), "action": "rollbar-report"})
				}
			case ln.PriCritical, ln.PriAlert, ln.PriEmergency:
				uid, err := client.Critical(err, extras)
				if err != nil {
					// These can't be Error or lnroll will recursively handle
					ln.Info(ln.F{"err": err, "uuid": uid, "priority": e.Pri.String(), "action": "rollbar-report"})
				}
			}
			return true
		}

		// Have a stack, so let's prepare it.
		st := sterr.StackTrace()

		var pc []uintptr

		// client wants a slice of uintptr, we have []errors.Frame, fix this
		for _, val := range st {
			pc = append(pc, uintptr(val))
		}

		switch e.Pri {
		case ln.PriError:
			uid, err := client.ErrorStack(err, pc, extras)
			if err != nil {
				// These can't be Error or lnroll will recursively handle
				ln.Info(ln.F{"err": err, "uuid": uid, "priority": e.Pri.String(), "action": "rollbar-report"})
			}
		case ln.PriCritical, ln.PriAlert, ln.PriEmergency:
			uid, err := client.CriticalStack(err, pc, extras)
			if err != nil {
				// These can't be Error or lnroll will recursively handle
				ln.Info(ln.F{"err": err, "uuid": uid, "priority": e.Pri.String(), "action": "rollbar-report"})
			}
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
