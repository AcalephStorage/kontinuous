package kv

import (
	"errors"
	"net"
	"strconv"
	"time"

	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/context"

	etcd "github.com/coreos/etcd/client"

	"github.com/AcalephStorage/kontinuous/util"
)

var kvLog = util.NewContextLogger("store/kv")

// KVClient holds the etcd client instance
type etcdClient struct {
	client etcd.KeysAPI
}

type KVClient interface {
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

// NewKVClient instantiates and establish connection to etcd
func NewEtcdClient(cacert, cert, key string, addresses ...string) (KVClient, error) {
	config, err := createKvConfig(cacert, cert, key, addresses)
	log := kvLog.InFunc("NewKVClient")

	if err != nil {
		log.WithError(err).Error("unable to create etcd client config")
		return nil, err
	}

	etc, err := etcd.New(*config)
	if err != nil {
		log.WithError(err).Error("unable to create kvclient.")
		return nil, err
	}

	return &etcdClient{etcd.NewKeysAPI(etc)}, nil
}

func (kv *etcdClient) Put(key, value string) error {
	_, err := kv.client.Set(context.Background(), key, value, &etcd.SetOptions{})
	return err
}

func (kv *etcdClient) Get(key string) (string, error) {
	res, err := kv.client.Get(context.Background(), key, &etcd.GetOptions{Quorum: true})
	if err != nil {
		return "", err
	}
	return res.Node.Value, nil
}

func (kv *etcdClient) PutInt(key string, value int) error {
	return kv.Put(key, strconv.Itoa(value))
}

func (kv *etcdClient) GetInt(key string) (int, error) {
	val, err := kv.Get(key)
	if err != nil {
		return -1, err
	}
	return strconv.Atoi(val)
}

func (kv *etcdClient) GetDir(key string) ([]*KVPair, error) {
	getOpts := &etcd.GetOptions{
		Quorum:    true,
		Recursive: true,
		Sort:      true,
	}

	res, err := kv.client.Get(context.Background(), key, getOpts)
	if err != nil {
		return nil, err
	}

	kvpair := []*KVPair{}
	for _, n := range res.Node.Nodes {
		kvpair = append(kvpair, &KVPair{
			Key:       n.Key,
			Value:     []byte(n.Value),
			LastIndex: n.ModifiedIndex,
		})
	}
	return kvpair, nil
}

func (kv *etcdClient) PutDir(key string) error {
	_, err := kv.client.Set(context.Background(), key, "", &etcd.SetOptions{Dir: true})
	return err
}

func (kv *etcdClient) PutIntDir(key string, value int) error {
	dirName := key + "/" + strconv.Itoa(value)
	return kv.PutDir(dirName)
}

func (kv *etcdClient) DeleteTree(key string) error {
	_, err := kv.client.Delete(context.Background(), key, &etcd.DeleteOptions{Recursive: true})
	return err
}

// utils

type certConfig struct {
	caCert string
	cert   string
	key    string
}

func createKvConfig(cacert, cert, key string, addresses []string) (*etcd.Config, error) {
	hasCA := cacert != ""
	hasCert := cert != ""
	hasKey := key != ""

	// default config
	var scheme = "http"
	config := &etcd.Config{
		Transport:               etcd.DefaultTransport,
		HeaderTimeoutPerRequest: 5 * time.Second,
	}

	if hasCA || hasCert || hasKey {
		certCfg := &certConfig{
			caCert: cacert,
			cert:   cert,
			key:    key,
		}

		ca, err := certCfg.loadCa()
		if err != nil {
			return nil, err
		}

		c, err := certCfg.loadCert()
		if err != nil {
			return nil, err
		}

		scheme = "https"
		setTLS(config, c, ca)
	}

	createEndpoints(config, scheme, addresses)

	return config, nil
}

func (cfg *certConfig) loadCa() (*x509.CertPool, error) {

	if cfg.caCert != "" {
		capem, err := ioutil.ReadFile(cfg.caCert)
		if err != nil {
			return nil, err
		}

		if ca := x509.NewCertPool(); ca.AppendCertsFromPEM(capem) {
			return ca, nil
		}
	}

	return nil, errors.New("unable to load certificate authority")
}

func (cfg *certConfig) loadCert() (tls.Certificate, error) {
	var err error
	var c tls.Certificate
	if cfg.cert != "" && cfg.key != "" {
		if c, err := tls.LoadX509KeyPair(cfg.cert, cfg.key); err == nil {
			return c, nil
		}
	}
	return c, err
}

func setTLS(config *etcd.Config, c tls.Certificate, ca *x509.CertPool) {
	// tls
	tlsCfg := &tls.Config{
		RootCAs:      ca,
		Certificates: []tls.Certificate{c},
	}

	config.Transport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tlsCfg,
	}
}

func createEndpoints(config *etcd.Config, scheme string, addresses []string) {
	var endpoints []string
	for _, addr := range addresses {
		endpoints = append(endpoints, scheme+"://"+addr)
	}
	config.Endpoints = endpoints
}
