package safeauth

import (
	"context"
	"log"
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
		log.Println("Check called before Grant")
		if reportOnly {
			return context.WithValue(c, checkedKey{}, true), true
		} else {
			return c, false
		}
	}
	for _, p := range ps {
		if !slices.Contains(granted, p) {
			log.Println("Check failed for " + p)
			if reportOnly {
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
			return true
		} else {
			return false
		}
	}
	return true
}
