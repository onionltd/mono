package oniontree

import (
	"github.com/oniontree-org/go-oniontree/validator"
	"github.com/oniontree-org/go-oniontree/validator/jsonschema"
	"regexp"
	"strings"
)

type serviceID string

func (i serviceID) String() string {
	return string(i)
}

func (i serviceID) Validate() error {
	pattern := `^[a-z0-9\-]+$`
	matched, err := regexp.MatchString(pattern, string(i))
	if err != nil {
		return err
	}
	if !matched {
		return &ErrInvalidID{string(i), pattern}
	}
	return nil
}

type Service struct {
	Name        string       `json:"name" yaml:"name"`
	Description string       `json:"description,omitempty" yaml:"description,omitempty"`
	URLs        []string     `json:"urls" yaml:"urls"`
	PublicKeys  []*PublicKey `json:"public_keys,omitempty" yaml:"public_keys,omitempty"`

	id        serviceID
	validator *validator.Validator
}

func (s *Service) ID() string {
	return string(s.id)
}

func (s *Service) SetURLs(urls []string) int {
	s.URLs = []string{}
	return s.AddURLs(urls)
}

func (s *Service) AddURLs(urls []string) int {
	urlExists := func(url string) bool {
		for idx, _ := range s.URLs {
			if s.URLs[idx] == url {
				return true
			}
		}
		return false
	}
	added := 0
	for _, url := range urls {
		url = strings.TrimSpace(url)
		if urlExists(url) {
			continue
		}
		s.URLs = append(s.URLs, url)
		added++
	}
	return added
}

func (s *Service) SetPublicKeys(publicKeys []*PublicKey) int {
	s.PublicKeys = []*PublicKey{}
	return s.AddPublicKeys(publicKeys)
}

func (s *Service) AddPublicKeys(publicKeys []*PublicKey) int {
	publicKeyExists := func(publicKey *PublicKey) bool {
		for idx, _ := range s.PublicKeys {
			if s.PublicKeys[idx].Fingerprint == "" && s.PublicKeys[idx].ID == "" {
				continue
			}
			if s.PublicKeys[idx].Fingerprint == publicKey.Fingerprint || s.PublicKeys[idx].ID == publicKey.ID {
				return true
			}
		}
		return false
	}
	added := 0
	for _, publicKey := range publicKeys {
		if publicKeyExists(publicKey) {
			continue
		}
		s.PublicKeys = append(s.PublicKeys, publicKey)
		added++
	}
	return added
}

func (s *Service) Validate() error {
	if err := s.id.Validate(); err != nil {
		return err
	}
	if s.validator != nil {
		return s.validator.Validate(s)
	}
	return nil
}

func NewService(id string) *Service {
	return &Service{
		id:        serviceID(id),
		validator: validator.NewValidator(jsonschema.V0),
	}
}
