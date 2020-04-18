package badger

import "time"

type KVPairInterface interface {
	Key() Key
	Value() ([]byte, error)
	Meta() byte
	Expires() time.Time

	SetKey(Key)
	SetValue([]byte) error
	SetMeta(byte)
	SetExpires(time.Time)

	Error() string
}
