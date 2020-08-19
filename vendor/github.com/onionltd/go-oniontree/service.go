package oniontree

import (
	"github.com/onionltd/go-oniontree/validator"
	"strings"
)

type Service struct {
	Name        string       `json:"name" yaml:"name"`
	Description string       `json:"description,omitempty" yaml:"description,omitempty"`
	URLs        []string     `json:"urls" yaml:"urls"`
	PublicKeys  []*PublicKey `json:"public_keys,omitempty" yaml:"public_keys,omitempty"`

	id        string
	validator *validator.Validator
}

func (s *Service) ID() string {
	return s.id
}

func (s *Service) SetURLs(urls []string) int {
	s.URLs = []string{}
	return s.AddURLs(urls)
}

func (s *Service) AddURLs(urls []string) int {
	added := 0
	for _, url := range urls {
		url = strings.TrimSpace(url)
		_, exists := s.urlExists(url)
		if exists {
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
	added := 0
	for _, publicKey := range publicKeys {
		idx, exists := s.publicKeyExists(publicKey)
		if exists {
			s.PublicKeys[idx] = publicKey
			continue
		}
		s.PublicKeys = append(s.PublicKeys, publicKey)
		added++
	}
	return added
}

func (s *Service) SetValidator(v *validator.Validator) {
	s.validator = v
}

func (s *Service) Validate() error {
	if s.validator == nil {
		return nil
	}
	return s.validator.Validate(s)
}

func (s Service) urlExists(url string) (int, bool) {
	for idx, _ := range s.URLs {
		if s.URLs[idx] == url {
			return idx, true
		}
	}
	return -1, false
}

func (s Service) publicKeyExists(publicKey *PublicKey) (int, bool) {
	for idx, _ := range s.PublicKeys {
		if s.PublicKeys[idx].Fingerprint == "" && s.PublicKeys[idx].ID == "" {
			continue
		}
		if s.PublicKeys[idx].Fingerprint == publicKey.Fingerprint || s.PublicKeys[idx].ID == publicKey.ID {
			return idx, true
		}
	}
	return -1, false
}

func NewService(id string) *Service {
	return &Service{
		id: id,
	}
}

func NewServiceWithValidator(id string, v *validator.Validator) *Service {
	s := NewService(id)
	s.validator = v
	return s
}
