package kube

import (
	"bytes"
	"fmt"
	"io"

	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

// JobClient is the interface to access the kubernetes job API
type JobClient interface {
	CreateJob(job *Job) error
	GetSecret(namespace string, secretName string) (map[string]string, error)
}

// concrete implementation of a job client
type realJobClient struct {
	*http.Client
	address string
	token   string
}

// NewClient returns a new JobClient connecting to the address. This uses the service
// account credentials
func NewClient(address string) (JobClient, error) {
	// create tls client
	cacertFile := "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	capem, err := ioutil.ReadFile(cacertFile)
	if err != nil {
		return nil, err
	}
	cacert := x509.NewCertPool()
	if !cacert.AppendCertsFromPEM(capem) {
		return nil, fmt.Errorf("unable to load certificate authority")
	}
	config := &tls.Config{RootCAs: cacert}
	transport := &http.Transport{TLSClientConfig: config}

	// read token
	client := &http.Client{Transport: transport}
	tokenFile := "/var/run/secrets/kubernetes.io/serviceaccount/token"
	token, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}
	return &realJobClient{client, address, string(token)}, nil
}

// Create a new kubernetes Job with the given job
func (r *realJobClient) CreateJob(job *Job) error {
	url := "/apis/extensions/v1beta1/namespaces/" + job.Metadata["namespace"].(string) + "/jobs"
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	byteData := bytes.NewReader(data)
	return r.doPost(url, byteData)

}

//Get secret with a given namespace and secret name
func (r *realJobClient) GetSecret(namespace string, secretName string) (map[string]string, error) {
	secret := &Secret{}
	secrets := make(map[string]string)

	uri := "/api/v1/namespaces/" + namespace + "/secrets/" + secretName

	err := r.doGet(uri, secret)
	if err != nil {
		return nil, err
	}

	for key, value := range secret.Data {
		decodedValue, err := base64.StdEncoding.DecodeString(value)

		if err != nil {
			logrus.WithError(err).Println("Unable to decode secret", key, value)
			continue
		}
		secrets[key] = string(decodedValue)
	}
	return secrets, nil
}

func (r *realJobClient) doGet(uri string, response interface{}) error {
	req, err := r.createRequest("GET", uri, nil)
	if err != nil {
		return err
	}
	res, err := r.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("%d: %s", res.StatusCode, string(body))
	}
	err = json.Unmarshal(body, response)
	if err != nil {
		return err
	}
	return nil
}

func (r *realJobClient) doPost(uri string, data io.Reader) error {
	req, err := r.createRequest("POST", uri, data)
	if err != nil {
		return err
	}
	res, err := r.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusCreated || res.StatusCode == http.StatusOK {
		return nil
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf("%d: %s", res.StatusCode, string(body))
}

func (r *realJobClient) createRequest(method, uri string, data io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, r.address+uri, data)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+r.token)
	return req, nil
}
