package lnroll

import (
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/apg/ln"
	"github.com/stvp/roll"
)

// New returns a new FilterFunc which reports errors to Rollbar.
func New() ln.FilterFunc {
	return ln.FilterFunc(func(e ln.Event) bool {
		if e.Pri < ln.PriError {
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

		// grab a list of pointers to all of the functions in the callstack
		pc := []uintptr{}
		runtime.Callers(1, pc)

		switch e.Pri {
		case ln.PriError:
			uid, err := roll.ErrorStack(err, pc, extras)
			if err != nil {
				// These can't be Error or lnroll will recursively handle
				ln.Info(ln.F{"err": err, "uuid": uid, "priority": e.Pri.String(), "action": "rollbar-report"})
			}
		case ln.PriCritical, ln.PriAlert, ln.PriEmergency:
			uid, err := roll.CriticalStack(err, pc, extras)
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
