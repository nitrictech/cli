package pflagext

import (
	"fmt"
	"strings"
)

type stringEnum struct {
	Allowed []string
	ValueP  *string
}

// newEnum give a list of allowed flag parameters, where the second argument is the default
func NewStringEnumVar(value *string, allowed []string, d string) *stringEnum {
	*value = d
	return &stringEnum{
		Allowed: allowed,
		ValueP:  value,
	}
}

func (e *stringEnum) String() string {
	return *e.ValueP
}

func (e *stringEnum) Set(p string) error {
	isIncluded := func(opts []string, val string) bool {
		for _, opt := range opts {
			if val == opt {
				return true
			}
		}
		return false
	}
	if !isIncluded(e.Allowed, p) {
		return fmt.Errorf("%s is not included in %s", p, strings.Join(e.Allowed, ","))
	}
	*e.ValueP = p
	return nil
}

func (e *stringEnum) Type() string {
	return "stringEnumVar"
}
