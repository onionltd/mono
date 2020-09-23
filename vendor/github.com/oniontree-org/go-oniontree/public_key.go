package oniontree

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/openpgp"
	"unicode"
)

type PublicKey struct {
	ID          string `json:"id,omitempty" yaml:"id,omitempty"`
	UserID      string `json:"user_id,omitempty" yaml:"user_id,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty" yaml:"fingerprint,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Value       string `json:"value" yaml:"value"`
}

func NewPublicKey(b []byte) (*PublicKey, error) {
	bClean := bytes.TrimLeftFunc(b, unicode.IsSpace)
	el, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(bClean))
	if err != nil {
		return nil, err
	}

	publicKey := &PublicKey{}
	for _, e := range el {
		userID := ""
		for _, ident := range e.Identities {
			userID = ident.Name
		}
		pk := e.PrimaryKey
		publicKey = &PublicKey{
			Value:       string(bClean),
			ID:          pk.KeyIdString(),
			Fingerprint: fmt.Sprintf("%X", pk.Fingerprint),
			UserID:      userID,
		}
	}
	return publicKey, nil
}
