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

	etcd "github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

// KVClient holds the etcd client instance
type EtcdClient struct {
	client etcd.KeysAPI
}

// NewKVClient instantiates and establish connection to etcd
func NewEtcdClient(cacert, cert, key string, addresses ...string) (Client, error) {
	config, err := createKvConfig(cacert, cert, key, addresses)

	if err != nil {
		return nil, err
	}

	etc, err := etcd.New(*config)
	if err != nil {
		return nil, err
	}

	return &EtcdClient{etcd.NewKeysAPI(etc)}, nil
}

func (kv *EtcdClient) Put(key, value string) error {
	_, err := kv.client.Set(context.Background(), key, value, &etcd.SetOptions{})
	return err
}

func (kv *EtcdClient) Get(key string) (string, error) {
	res, err := kv.client.Get(context.Background(), key, &etcd.GetOptions{Quorum: true})
	if err != nil {
		return "", err
	}
	return res.Node.Value, nil
}

func (kv *EtcdClient) PutInt(key string, value int) error {
	return kv.Put(key, strconv.Itoa(value))
}

func (kv *EtcdClient) GetInt(key string) (int, error) {
	val, err := kv.Get(key)
	if err != nil {
		return -1, err
	}
	return strconv.Atoi(val)
}

func (kv *EtcdClient) GetDir(key string) ([]*KVPair, error) {
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

func (kv *EtcdClient) PutDir(key string) error {
	_, err := kv.client.Set(context.Background(), key, "", &etcd.SetOptions{Dir: true})
	return err
}

func (kv *EtcdClient) PutIntDir(key string, value int) error {
	dirName := key + "/" + strconv.Itoa(value)
	return kv.PutDir(dirName)
}

func (kv *EtcdClient) DeleteTree(key string) error {
	_, err := kv.client.Delete(context.Background(), key, &etcd.DeleteOptions{Recursive: true})
	return err
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
