package badger

import "time"

type KVPair struct {
	key     Key
	value   []byte
	meta    byte
	expires time.Time
}

func (kv KVPair) Key() Key {
	return kv.key
}

func (kv KVPair) Value() ([]byte, error) {
	return kv.value, nil
}

func (kv KVPair) Meta() byte {
	return kv.meta
}

func (kv KVPair) Expires() time.Time {
	return kv.expires
}

func (kv *KVPair) SetKey(k Key) {
	kv.key = k
}

func (kv *KVPair) SetValue(v []byte) error {
	kv.value = v
	return nil
}

func (kv *KVPair) SetMeta(m byte) {
	kv.meta = m
}

func (kv *KVPair) SetExpires(t time.Time) {
	kv.expires = t
}

func (kv KVPair) Error() string {
	return ""
}
