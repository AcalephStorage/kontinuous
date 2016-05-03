package kube

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"net/http"
)

// KubeClient is the interface to access the kubernetes job API
type KubeClient interface {
	CreateJob(job *Job) error
	GetSecret(namespace string, secretName string) (map[string]string, error)
	DeployResourceFile(resourceFile []byte) error
}

// concrete implementation of a job client
type realKubeClient struct {
	*http.Client
	address string
	token   string
}

// NewClient returns a new KubeClient connecting to the address. This uses the service
// account credentials
func NewClient(address string) (KubeClient, error) {
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
	return &realKubeClient{client, address, string(token)}, nil
}

// Create a new kubernetes Job with the given job
func (r *realKubeClient) CreateJob(job *Job) error {
	url := "/apis/extensions/v1beta1/namespaces/" + job.Metadata["namespace"].(string) + "/jobs"
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}
	byteData := bytes.NewReader(data)
	return r.doPost(url, byteData)

}

// DeployResourceFile deploys a Kubernbetes YAML spec file as Kubernetes resources
func (r *realKubeClient) DeployResourceFile(resourceFile []byte) error {

	resources := strings.Split(string(resourceFile), "---")

	for _, resource := range resources {

		// if it's empty, skip
		if strings.TrimSpace(resource) == "" {
			continue
		}

		logrus.Info("deploying to kubernetes: ", resource)

		data, err := yaml.YAMLToJSON([]byte(resource))
		if err != nil {
			logrus.WithError(err).Error("unable to convert yaml to json")
			return err
		}

		var out map[string]interface{}
		err = json.Unmarshal(data, &out)
		if err != nil {
			logrus.WithError(err).Error("unable to unmarshal json to map")
			return err
		}

		// if unmarshalled data is nil, skip
		if out == nil {
			continue
		}

		kind := strings.ToLower(out["kind"].(string)) + "s"
		metadata := out["metadata"]
		namespace := "default"
		name := ""
		if metadata != nil {
			namespace = metadata.(map[string]interface{})["namespace"].(string)
			name = metadata.(map[string]interface{})["name"].(string)
		}

		// endpoint is /api/v1/namespaces/{namespace}/{resourceType}
		uri := fmt.Sprintf("/api/v1/namespaces/%s/%s/%s", namespace, kind, name)

		err = r.doGet(uri, &out)
		if out != nil {
			err = r.doDelete(uri)
			time.Sleep(30 * time.Second)
			if err != nil {
				logrus.WithError(err).Error("unable to DELETE resource")
			}
		}

		postUri := fmt.Sprintf("/api/v1/namespaces/%s/%s", namespace, kind)
		err = r.doPost(postUri, bytes.NewReader(data))
		if err != nil {
			logrus.WithError(err).Error("unable to POST data")
			return err
		}
	}

	return nil
}

//Get secret with a given namespace and secret name
func (r *realKubeClient) GetSecret(namespace string, secretName string) (map[string]string, error) {
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

func (r *realKubeClient) doGet(uri string, response interface{}) error {
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

func (r *realKubeClient) doDelete(uri string) error {

	req, err := r.createRequest("DELETE", uri, nil)
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

	return nil
}

func (r *realKubeClient) doPost(uri string, data io.Reader) error {
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

func (r *realKubeClient) createRequest(method, uri string, data io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, r.address+uri, data)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+r.token)
	return req, nil
}
