package quadlek

import "github.com/boltdb/bolt"

type Store struct {
	db       *bolt.DB
	pluginId string
}

func (s *Store) Update(key string, value []byte) error {
	err := s.db.Update(func(tx *bolt.Tx) error {
		rootBkt, err := tx.CreateBucketIfNotExists([]byte("plugins"))
		if err != nil {
			return err
		}

		pluginBkt, err := rootBkt.CreateBucketIfNotExists([]byte(s.pluginId))
		if err != nil {
			return err
		}

		err = pluginBkt.Put([]byte(key), value)
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
		rootBkt, err := tx.CreateBucketIfNotExists([]byte("plugins"))
		if err != nil {
			return err
		}

		println(s.pluginId)
		pluginBkt, err := rootBkt.CreateBucketIfNotExists([]byte(s.pluginId))
		if err != nil {
			return err
		}

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

func (s *Store) Get(key string, getFunc func([]byte)) error {
	err := s.db.View(func(tx *bolt.Tx) error {
		stringKey := []byte(key)
		rootBkt, err := tx.CreateBucketIfNotExists([]byte("plugins"))
		if err != nil {
			return err
		}

		pluginBkt, err := rootBkt.CreateBucketIfNotExists([]byte(s.pluginId))
		if err != nil {
			return err
		}

		val := pluginBkt.Get(stringKey)
		getFunc(val)

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
