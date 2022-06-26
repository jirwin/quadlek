package boltdb

import "github.com/boltdb/bolt"

type PluginStore interface {
	ForEach(forEachFunc func(bucket *bolt.Bucket, key string, value []byte) error) error
	Update(key string, value []byte) error
	UpdateRaw(updateFunc func(*bolt.Bucket) error) error
	GetAndUpdate(key string, updateFunc func([]byte) ([]byte, error)) error
	Get(key string, getFunc func([]byte) error) error
}

type pluginStore struct {
	pluginID string
	db       *bolt.DB
}

func (s *pluginStore) ForEach(forEachFunc func(bucket *bolt.Bucket, key string, value []byte) error) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		rootBkt := tx.Bucket([]byte("plugins"))

		pluginBkt := rootBkt.Bucket([]byte(s.pluginID))
		err := pluginBkt.ForEach(func(k []byte, v []byte) error {
			return forEachFunc(pluginBkt, string(k), v)
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// Update stores the value at the provided key
func (s *pluginStore) Update(key string, value []byte) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		rootBkt := tx.Bucket([]byte("plugins"))

		pluginBkt := rootBkt.Bucket([]byte(s.pluginID))

		err := pluginBkt.Put([]byte(key), value)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// UpdateRaw allows you direct access to the database when the simple key value interface doesn't work.
func (s *pluginStore) UpdateRaw(updateFunc func(*bolt.Bucket) error) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		rootBkt := tx.Bucket([]byte("plugins"))

		pluginBkt := rootBkt.Bucket([]byte(s.pluginID))

		err := updateFunc(pluginBkt)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// GetAndUpdate retrieves a key from the database and passes its value to the provided updateFunc.
// This allows you to transform data atomically.
func (s *pluginStore) GetAndUpdate(key string, updateFunc func([]byte) ([]byte, error)) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		stringKey := []byte(key)
		rootBkt := tx.Bucket([]byte("plugins"))

		pluginBkt := rootBkt.Bucket([]byte(s.pluginID))

		val := pluginBkt.Get(stringKey)
		updateVal, err := updateFunc(val)
		if err != nil {
			return err
		}

		if updateVal != nil {
			err = pluginBkt.Put(stringKey, updateVal)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// Get retrieves a key from the database and passes it to the provided getFunc
func (s *pluginStore) Get(key string, getFunc func([]byte) error) error {
	err := s.db.View(func(tx *bolt.Tx) error {
		stringKey := []byte(key)
		rootBkt := tx.Bucket([]byte("plugins"))

		pluginBkt := rootBkt.Bucket([]byte(s.pluginID))

		val := pluginBkt.Get(stringKey)
		return getFunc(val)
	})
	if err != nil {
		return err
	}

	return nil
}
