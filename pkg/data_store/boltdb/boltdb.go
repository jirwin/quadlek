package boltdb

import (
	"fmt"
	"github.com/boltdb/bolt"
	"go.uber.org/zap"
	"os"
	"time"
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

	return c, nil
}

type BoltDbStore struct {
	C  Config
	L  *zap.Logger
	db *bolt.DB
}

func (b *BoltDbStore) Close() {
	if b.db != nil {
		b.db.Close()
	}
}

func New(c Config, l *zap.Logger) (*BoltDbStore, error) {
	b := &BoltDbStore{
		C: c,
		L: l,
	}

	db, err := bolt.Open(c.DbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	b.db = db

	return b, nil
}
