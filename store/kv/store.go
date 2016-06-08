package kv

type Client interface {
	// Put sets the value of a key
	Put(key, value string) error

	// Get returns the value of the specified key.
	Get(key string) (string, error)

	// PutInt accepts an Int value and store it under the specified key.
	PutInt(key string, value int) error

	// GetInt returns the value of the specified key. In Int type
	GetInt(key string) (int, error)

	// GetDir returns the child nodes of a given directory
	GetDir(key string) ([]*KVPair, error)

	// PutDir creates a directory.
	PutDir(key string) error

	// PutIntDir creates an integer directory under the given key
	PutIntDir(key string, value int) error

	// DeleteTree removes a reange of keys under the given directory.
	DeleteTree(key string) error
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
