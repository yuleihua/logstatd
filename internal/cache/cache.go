package cache

import (
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type CacheData struct {
	DB       *leveldb.DB
	FileName string
	Opt      *opt.Options
}

func NewCacheData(file string) *CacheData {
	opt := &opt.Options{
		BlockSize: 8 * 1024,
	}
	db, err := leveldb.OpenFile(file, opt)
	if err != nil {
		log.Fatalf("Open leveldb error,%v:%v", file, err)
	}

	return &CacheData{
		FileName: file,
		DB:       db,
		Opt:      opt,
	}
}

func (c *CacheData) Get(key []byte) ([]byte, error) {
	return c.DB.Get(key, nil)
}

func (c *CacheData) Has(key []byte) (bool, error) {
	return c.DB.Has(key, nil)
}

func (c *CacheData) Put(key, val []byte) error {
	return c.DB.Put(key, val, nil)
}

func (c *CacheData) Delete(key []byte) error {
	return c.DB.Delete(key, nil)
}

func (c *CacheData) Close() error {
	return c.DB.Close()
}

func (c *CacheData) RunAll(fn func(string, string) error, isStop bool) error {
	if fn == nil {
		return nil
	}

	iter := c.DB.NewIterator(nil, nil)
	defer iter.Release()

	for i := 0; iter.Next(); i++ {
		k := iter.Key()
		v := iter.Value()
		// fmt.Printf("i k v, %d:%q:%q", i, k, v)
		if err := fn(string(k), string(v)); err != nil && isStop {
			return err
		}
	}

	if err := iter.Error(); err != nil {
		log.Errorf("error:%v", err)
		return err
	}
	return nil
}

type CacheDataTx struct {
	Tr *leveldb.Transaction
}

func (c *CacheData) BeginTx() (*CacheDataTx, error) {
	tr, err := c.DB.OpenTransaction()
	if err != nil {
		return nil, err
	}

	return &CacheDataTx{tr}, nil
}

func (t *CacheDataTx) Get(key []byte) ([]byte, error) {
	return t.Tr.Get(key, nil)
}

func (t *CacheDataTx) Put(key, val []byte) error {
	return t.Tr.Put(key, val, nil)
}

func (t *CacheDataTx) Delete(key []byte) error {
	return t.Tr.Delete(key, nil)
}

func (t *CacheDataTx) Has(key []byte) (bool, error) {
	return t.Tr.Has(key, nil)
}

func (t *CacheDataTx) Commit() error {
	return t.Tr.Commit()
}

func (t *CacheDataTx) Discard() error {
	t.Tr.Discard()
	return nil
}
