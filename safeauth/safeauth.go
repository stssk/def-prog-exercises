package safeauth

import (
	"context"
	"log"
	"runtime/debug"
	"slices"
)

var reportOnly bool

func SetReportOnly() {
	reportOnly = true
}

type privilegeKey struct{}

func Grant(c context.Context, ps ...string) context.Context {
	if c.Value(privilegeKey{}) != nil {
		if reportOnly {
			debug.PrintStack()
			log.Println("Grant called multiple times")
		} else {
			panic("Grant called multiple times")
		}
	}
	return context.WithValue(c, privilegeKey{}, ps)
}

type checkedKey struct{}

func Check(c context.Context, ps ...string) (_ context.Context, ok bool) {
	granted, ok := c.Value(privilegeKey{}).([]string)
	if !ok {
		if reportOnly {
			log.Println("Check called before Grant")
			debug.PrintStack()
			return context.WithValue(c, checkedKey{}, true), true
		} else {
			return c, false
		}
	}
	for _, p := range ps {
		if !slices.Contains(granted, p) {
			if reportOnly {
				log.Println("Check failed for " + p)
				debug.PrintStack()
				continue
			} else {
				return c, false
			}
		}
	}
	return context.WithValue(c, checkedKey{}, true), true
}

func Must(c context.Context) (ok bool) {
	if c.Value(checkedKey{}) == nil {
		log.Println("Must called before Check")
		if reportOnly {
			debug.PrintStack()
			return true
		} else {
			return false
		}
	}
	return true
}
