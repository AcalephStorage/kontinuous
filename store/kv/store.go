package kv

type Client interface {
	Create(key string, value []byte) error

	CreateDir(name string) error

	CreateInDir(dir string, value []byte) (key string, err error)

	Restore(key string) ([]byte, error)

	Update(key string, value []byte) error

	Delete(key string) error

	List(dir string) ([][]byte, error)
}

type WatchResponse struct {
	action string
	key    string
	value  []byte
}

// KVPair defines the retrieved key and value
type KVPair struct {
	Key       string
	Value     []byte
	LastIndex uint64
}

// utils

type certConfig struct {
	caCert string
	cert   string
	key    string
}
