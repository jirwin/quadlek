package boltdb

import (
	"fmt"
	"os"
	"time"

	"github.com/boltdb/bolt"
	"go.uber.org/zap"
)

type Config struct {
	DbPath string
}

func NewConfig() (Config, error) {
	c := Config{}

	dbPath := os.Getenv("QUADLEK_DB_PATH")
	if dbPath == "" {
		return Config{}, fmt.Errorf("QUADLEK_DB_PATH must be set")
	}
	c.DbPath = dbPath
	return c, nil
}

type BoltDbStore struct {
	c  Config
	l  *zap.Logger
	db *bolt.DB
}

func (b *BoltDbStore) InitPluginBucket(pluginID string) error {
	err := b.db.Update(func(tx *bolt.Tx) error {
		rootBkt, err := tx.CreateBucketIfNotExists([]byte("plugins"))
		if err != nil {
			return err
		}

		_, err = rootBkt.CreateBucketIfNotExists([]byte(pluginID))
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

func (b *BoltDbStore) GetStore(pluginID string) PluginStore {
	return &pluginStore{
		pluginID: pluginID,
		db:       b.db,
	}
}

func (b *BoltDbStore) Close() {
	if b.db != nil {
		b.db.Close()
		b.db = nil
	}
}

func New(c Config, l *zap.Logger) (*BoltDbStore, error) {
	b := &BoltDbStore{
		c: c,
		l: l.Named("boltdb-datastore"),
	}

	db, err := bolt.Open(c.DbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	b.db = db

	return b, nil
}
