package service

import "strings"

type Service struct {
	ID          string      `json:"-" yaml:"-"`
	Name        string      `json:"name" yaml:"name"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	URLs        []string    `json:"urls" yaml:"urls"`
	PublicKeys  []PublicKey `json:"public_keys,omitempty" yaml:"public_keys,omitempty"`
}

func (s *Service) SetURLs(urls ...string) {
	s.URLs = []string{}
	s.AddURLs(urls...)
}

func (s *Service) AddURLs(urls ...string) {
	for _, url := range urls {
		url = strings.TrimSpace(url)
		_, exists := s.urlExists(url)
		if exists {
			continue
		}
		s.URLs = append(s.URLs, url)
	}
}

func (s *Service) AddPublicKeys(publicKeys ...PublicKey) {
	for _, publicKey := range publicKeys {
		idx, exists := s.publicKeyExists(publicKey)
		if exists {
			s.PublicKeys[idx] = publicKey
			continue
		}
		s.PublicKeys = append(s.PublicKeys, publicKey)
	}
}

func (s *Service) SetPublicKeys(publicKeys ...PublicKey) {
	s.PublicKeys = []PublicKey{}
	s.AddPublicKeys(publicKeys...)
}

func (s Service) urlExists(url string) (int, bool) {
	for idx, _ := range s.URLs {
		if s.URLs[idx] == url {
			return idx, true
		}
	}
	return -1, false
}

func (s Service) publicKeyExists(publicKey PublicKey) (int, bool) {
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
