package quadlek

import "github.com/boltdb/bolt"

type Store struct {
	db       *bolt.DB
	pluginId string
}

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
