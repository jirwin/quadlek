package quadlek

import "github.com/boltdb/bolt"

// The Store struct provides a plugin a namespaced key value store for the plugin to use however it needs.
// By default, keys are strings, and values are []byte. You can use UpdateRaw() if this doesn't fit your needs.
type Store struct {
	db       *bolt.DB
	pluginId string
}

// Update stores the value at the provided key
func (s *Store) Update(key string, value []byte) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		rootBkt := tx.Bucket([]byte("plugins"))

		pluginBkt := rootBkt.Bucket([]byte(s.pluginId))

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
func (s *Store) UpdateRaw(updateFunc func(*bolt.Bucket) error) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		rootBkt := tx.Bucket([]byte("plugins"))

		pluginBkt := rootBkt.Bucket([]byte(s.pluginId))

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
func (s *Store) GetAndUpdate(key string, updateFunc func([]byte) ([]byte, error)) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		stringKey := []byte(key)
		rootBkt := tx.Bucket([]byte("plugins"))

		pluginBkt := rootBkt.Bucket([]byte(s.pluginId))

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
func (s *Store) Get(key string, getFunc func([]byte) error) error {
	err := s.db.View(func(tx *bolt.Tx) error {
		stringKey := []byte(key)
		rootBkt := tx.Bucket([]byte("plugins"))

		pluginBkt := rootBkt.Bucket([]byte(s.pluginId))

		val := pluginBkt.Get(stringKey)
		return getFunc(val)
	})
	if err != nil {
		return err
	}

	return nil
}

// InitPluginBucket initializes the database bucket for the given pluginId
func (b *Bot) InitPluginBucket(pluginId string) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		rootBkt, err := tx.CreateBucketIfNotExists([]byte("plugins"))
		if err != nil {
			return err
		}

		_, err = rootBkt.CreateBucketIfNotExists([]byte(pluginId))
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
