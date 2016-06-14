package kv

import (
	"errors"
	"net"
	"time"

	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
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

func (kv *EtcdClient) Create(key string, value []byte) error {
	_, err := kv.client.Create(context.Background(), key, string(value))
	if err != nil {
		log.WithError(err).Debug("unable to create new etcd entry")
	}
	return err
}

func (kv *EtcdClient) CreateDir(name string) error {
	opts := &etcd.SetOptions{Dir: true}
	_, err := kv.client.Set(context.Background(), name, "", opts)
	if err != nil {
		log.WithError(err).Debug("unable to create new etcd directory")
	}
	return err
}

func (kv *EtcdClient) CreateInDir(dir string, value []byte) (key string, err error) {
	res, err := kv.client.CreateInOrder(context.Background(), dir, string(value), nil)
	if err != nil {
		log.WithError(err).Debugf("unable to create etcd entry in directory %s", dir)
		return
	}
	key = res.Node.Key
	return
}

func (kv *EtcdClient) Restore(key string) (value []byte, err error) {
	res, err := kv.client.Get(context.Background(), key, nil)
	if err != nil {
		log.WithError(err).Debugf("unable to retrieve etcd value for key %s", key)
		return
	}
	value = []byte(res.Node.Value)

	return
}

func (kv *EtcdClient) Update(key string, value []byte) error {
	// opts := &etcd.SetOptions{PrevExist: etcd.PrevExist}
	_, err := kv.client.Set(context.Background(), key, string(value), nil)
	if err != nil {
		log.WithError(err).Debugf("unable to update etcd entry with key %s", key)
	}
	return err
}

func (kv *EtcdClient) Delete(key string) error {
	opts := &etcd.DeleteOptions{Recursive: true}
	_, err := kv.client.Delete(context.Background(), key, opts)
	if err != nil {
		log.WithError(err).Debugf("unable to delete etcd entry with key %s", key)
	}

	return err
}

func (kv *EtcdClient) List(dir string) (values [][]byte, err error) {
	opts := &etcd.GetOptions{Recursive: true, Sort: true}
	res, err := kv.client.Get(context.Background(), dir, opts)
	if err != nil {
		log.WithError(err).Debugf("unable to list entries of directory %s", dir)
		return
	}
	nodes := res.Node.Nodes
	values = make([][]byte, len(nodes))
	for i, node := range nodes {
		values[i] = []byte(node.Value)
	}
	return
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
