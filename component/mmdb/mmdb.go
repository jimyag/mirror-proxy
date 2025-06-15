package mmdb

import (
	"sync"

	"github.com/oschwald/maxminddb-golang/v2"
)

var (
	once sync.Once
	mmdb *maxminddb.Reader
)

func LoadMMDB(path string) (*maxminddb.Reader, error) {
	var err error

	once.Do(func() {
		mmdb, err = maxminddb.Open(path)
		if err != nil {
			return
		}
	})
	if err != nil {
		return nil, err
	}
	return mmdb, nil
}
