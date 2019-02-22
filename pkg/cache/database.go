package cache

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/boltdb/bolt"
)

// DataBase is a wrapper struct for boltdb key-value pair database.
type DataBase struct {
	Path          string
	DefaultBucket string
	db            *bolt.DB
}

// GetDB returns an initialized DataBase. Will either create a brand new boltdb
// or open existing one.
func GetDB(dbfile string) (*DataBase, error) {
	name := filename(dbfile)
	boltdb, err := bolt.Open(dbfile, 0600, nil)
	if err != nil {
		return nil, err
	}

	err = boltdb.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(name))
		return err
	})
	db := &DataBase{
		Path:          dbfile,
		DefaultBucket: name,
		db:            boltdb,
	}
	return db, err
}

// Put stores bytes the database
func (db *DataBase) Put(key string, val []byte) error {
	return db.update(func(b *bolt.Bucket) error {
		return b.Put([]byte(key), val)
	})
}

// Get will retrieve the value given a key
func (db *DataBase) Get(key string) (raw []byte, err error) {
	err = db.view(func(b *bolt.Bucket) error {
		raw = b.Get([]byte(key))
		return nil
	})
	return raw, err
}

// Exists will return true if the key supplied has data associated with it.
func (db *DataBase) Exists(key string) (exists bool) {
	if err := db.view(func(b *bolt.Bucket) error {
		data := b.Get([]byte(key))

		if data == nil {
			exists = false
		} else {
			exists = true
		}
		return nil
	}); err != nil {
		return false
	}
	return exists
}

// Close will close the DataBase's inner bolt.DB
func (db *DataBase) Close() error {
	return db.db.Close()
}

// Destroy will close the database and completly delete the database file.
func (db *DataBase) Destroy() error {
	err := db.Close()
	if err != nil {
		return err
	}
	return os.Remove(db.Path)
}

func (db *DataBase) view(fn func(*bolt.Bucket) error) error {
	return db.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(db.DefaultBucket))
		return fn(bucket)
	})
}

func (db *DataBase) update(fn func(*bolt.Bucket) error) error {
	return db.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(db.DefaultBucket))
		return fn(bucket)
	})
}

func filename(file string) string {
	fname := filepath.Base(file)
	return strings.TrimSuffix(fname, filepath.Ext(fname))
}
