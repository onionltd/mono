package config

import "strings"

type AuthString string

func (s AuthString) split() []string {
	return strings.SplitN(string(s), ":", 2)
}

func (s AuthString) Username() string {
	return s.split()[0]
}

func (s AuthString) Password() string {
	ss := s.split()
	if len(ss) < 2 {
		return ""
	}
	return ss[1]
}
