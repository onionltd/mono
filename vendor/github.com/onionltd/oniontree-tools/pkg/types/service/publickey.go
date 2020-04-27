package service

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/openpgp"
)

type PublicKey struct {
	ID          string `json:"id,omitempty" yaml:"id,omitempty"`
	UserID      string `json:"user_id,omitempty" yaml:"user_id,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty" yaml:"fingerprint,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Value       string `json:"value" yaml:"value"`
}

func ParseKey(key []byte) (PublicKey, error) {
	el, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(key))
	if err != nil {
		return PublicKey{}, err
	}

	publicKey := PublicKey{}
	for _, e := range el {
		userID := ""
		for _, ident := range e.Identities {
			userID = ident.Name
		}
		pk := e.PrimaryKey
		publicKey = PublicKey{
			Value:       string(key),
			ID:          pk.KeyIdString(),
			Fingerprint: fmt.Sprintf("%X", pk.Fingerprint),
			UserID:      userID,
		}
	}
	return publicKey, nil
}
