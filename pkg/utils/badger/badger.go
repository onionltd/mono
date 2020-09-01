package badger

import (
	"github.com/dgraph-io/badger/v2"
	"time"
)

type Key []byte

func Load(db *badger.DB, k Key, kv KVPairInterface) error {
	return db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(k)
		if err != nil {
			return err
		}
		v, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		kv.SetKey(k)
		kv.SetMeta(item.UserMeta())
		kv.SetExpires(time.Unix(int64(item.ExpiresAt()), 0))
		if err := kv.SetValue(v); err != nil {
			return err
		}
		return nil
	})
}

func Store(db *badger.DB, kv KVPairInterface) error {
	return db.Update(func(txn *badger.Txn) error {
		v, err := kv.Value()
		if err != nil {
			return err
		}
		return txn.SetEntry(&badger.Entry{
			Key:       kv.Key(),
			Value:     v,
			UserMeta:  kv.Meta(),
			ExpiresAt: uint64(kv.Expires().Unix()),
		})
	})
}
