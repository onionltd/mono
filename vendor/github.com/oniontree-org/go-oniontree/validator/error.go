package validator

import "strings"

type ValidatorError struct {
	errs []error
}

func (e *ValidatorError) Error() string {
	s := make([]string, 0, len(e.errs))
	for i := range e.errs {
		s = append(s, e.errs[i].Error())
	}
	return strings.Join(s, "\n")
}
