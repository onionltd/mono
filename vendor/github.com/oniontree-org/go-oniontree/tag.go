package oniontree

import "regexp"

type Tag string

func (t Tag) String() string {
	return string(t)
}

func (t Tag) Validate() error {
	pattern := `^[a-z0-9\-]+$`
	matched, err := regexp.MatchString(pattern, string(t))
	if err != nil {
		return err
	}
	if !matched {
		return &ErrInvalidTagName{string(t), pattern}
	}
	return nil
}
