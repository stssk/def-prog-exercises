package authentication

import (
	"context"
	"fmt"
	"net/http"
)

type Privilege string

type privilegeKey struct{}

func Grant(c context.Context, ps ...Privilege) context.Context {
	if (c.Value(privilegeKey{}) == nil) {
		panic("Grant called multiple times")
	}
	return context.WithValue(c, privilegeKey{}, ps)
}

type checkedKey struct{}

func Check(c context.Context, ps ...Privilege) (_ context.Context, err error) {
	if (c.Value(privilegeKey{}) == nil) {
		panic("Check called before Grant")
	}
	checked := false
	for _, p := range ps {
		for _, q := range c.Value(privilegeKey{}).([]Privilege) {
			if p == q {
				checked = true
				break
			}
		}
		if !checked {
			return c, fmt.Errorf("missing privilege %s", p)
		}
	}
	return context.WithValue(c, checkedKey{}, nil), nil
}

func Must(c context.Context) (err error) {
	if c.Value(checkedKey{}) == nil {
		return fmt.Errorf("missing privilege check")
	}
	return nil
}

func GrantRequestPrivileges(r *http.Request, ps ...Privilege) *http.Request {
	c := Grant(r.Context(), ps...)
	return r.WithContext(c)
}

func CheckRequestPrivileges(r *http.Request, ps ...Privilege) (*http.Request, error) {
	_, err := Check(r.Context(), ps...)
	if err != nil {
		return r, err
	}
	return r, nil
}

func GetRequestPrivileges(r *http.Request) []Privilege {
	return r.Context().Value(privilegeKey{}).([]Privilege)
}
