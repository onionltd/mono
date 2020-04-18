package links

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	badger2 "github.com/onionltd/mono/pkg/utils/badger"
	"time"
)

const (
	FingerprintLength = 5
	// FingerprintLimitBytes specifies how many bytes of the value are used to calculate the fingerprint.
	// It's here to prevent DoS where an attacker sends large URLs that needs to be hashed.
	FingerprintLimitBytes = 128
)

// linkBare is a structure that is actually stored in badger.
type linkBare struct {
	ServiceID string `json:"service_id"`
	Path      string `json:"path"`
}

type Link struct {
	fingerprint string
	serviceID   string
	path        string
}

func (l Link) Fingerprint() string {
	return l.fingerprint
}

func (l Link) ServiceID() string {
	return l.serviceID
}

func (l Link) Path() string {
	return l.path
}

//
// Methods to fulfill badger interface.
//
func (l Link) Key() badger2.Key {
	return badger2.Key(l.fingerprint)
}

func (l *Link) SetKey(k badger2.Key) {
	l.fingerprint = string(k)
}

func (l Link) Value() ([]byte, error) {
	return json.Marshal(linkBare{
		ServiceID: l.serviceID,
		Path:      l.path,
	})
}

func (l *Link) SetValue(v []byte) error {
	bare := linkBare{}
	if err := json.Unmarshal(v, &bare); err != nil {
		return err
	}
	l.serviceID = bare.ServiceID
	l.path = bare.Path
	return nil
}

func (l Link) Meta() byte { return 0 }

func (l Link) SetMeta(m byte) {}

func (l Link) Expires() time.Time { return time.Unix(0, 0) }

func (l *Link) SetExpires(t time.Time) {}

func (l Link) Error() string { return "" }

func NewLink(serviceID, path string) (*Link, error) {
	fingerprint := func(serviceID, url string) string {
		// TODO: optimize this part!
		// 	https://golang.org/pkg/strings/#Builder
		// TODO: trim `b` to constant length
		b := []byte(fmt.Sprintf("%s/%s", serviceID, url))
		if len(b) > FingerprintLimitBytes {
			b = b[:FingerprintLimitBytes]
		}
		sum := sha256.Sum256(b)
		return fmt.Sprintf("%x", sum[:FingerprintLength])
	}
	return &Link{
		fingerprint: fingerprint(serviceID, path),
		serviceID:   serviceID,
		path:        path,
	}, nil
}
