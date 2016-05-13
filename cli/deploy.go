package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"
)

var secretData = `
{
  "AuthSecret": "{{.AuthCode}}",
  "S3SecretKey": "{{.SecretKey}}",
  "S3AccessKey": "{{.AccessKey}}"
}

`

var minio = `
---
kind: Service
apiVersion: v1
metadata:
  name: minio
  namespace: {{.Namespace}}
  labels:
    app: minio
    type: object-store
spec:
  selector:
    app: minio
    type: object-store
  ports:
    - name: service
      port: 9000
      targetPort: 9000
---
kind: ReplicationController
apiVersion: v1
metadata:
  name: minio
  namespace: {{.Namespace}}
  labels:
    app: minio
    type: object-store
spec:
  replicas: 1
  selector:
    app: minio
    type: object-store
  template:
    metadata:
      name: minio
      labels:
        app: minio
        type: object-store
    spec:
      containers:
        - name: minio
          image: minio/minio:latest
          imagePullPolicy: Always
          env:
            - name: MINIO_ACCESS_KEY
              value: {{.AccessKey}}
            - name: MINIO_SECRET_KEY
              value: {{.SecretKey}}
          ports:
            - name: service
              containerPort: 9000
          livenessProbe:
            tcpSocket:
              port: 9000
            timeoutSeconds: 1
`

var secret = `

---
kind: Secret
apiVersion: v1
metadata:
  name: kontinuous-secrets
  namespace: {{.Namespace}}
data:
  kontinuous-secrets: {{.SecretData}}
  
`

var etcd = `
---
kind: Service
apiVersion: v1
metadata:
  name: etcd
  namespace: {{.Namespace}}
  labels:
    app: etcd
    type: kv
spec:
  selector:
    app: etcd
    type: kv
  ports:
    - name: api
      port: 2379
      targetPort: 2379
---
kind: ReplicationController
apiVersion: v1
metadata:
  name: etcd
  namespace: {{.Namespace}}
  labels:
    app: etcd
    type: kv
spec:
  replicas: 1
  selector:
    app: etcd
    type: kv
  template:
    metadata:
      labels:
        app: etcd
        type: kv
    spec:
      containers:
        - name: etcd
          image: quay.io/coreos/etcd:v2.2.2
          imagePullPolicy: Always
          args:
            - --listen-client-urls
            - http://0.0.0.0:2379
            - --advertise-client-urls
            - http://0.0.0.0:2379
          ports:
            - name: api
              containerPort: 2379
`

var registry = `
---
kind: Service
apiVersion: v1
metadata:
  name: registry
  namespace: {{.Namespace}}
  labels:
    app: registry
    type: storage
spec:
  selector:
    app: registry
    type: storage
  ports:
    - name: service
      port: 5000
      targetPort: 5000
---
kind: ReplicationController
apiVersion: v1
metadata:
  name: registry
  namespace: {{.Namespace}}
  labels:
    app: registry
    type: storage
spec:
  replicas: 1
  selector:
    app: registry
    type: storage
  template:
    metadata:
      name: registry
      namespace: {{.Namespace}}
      labels:
        app: registry
        type: storage
    spec:
      containers:
        - name: registry
          image: registry:2
          ports:
            - name: service
              containerPort: 5000

`

var kontinuousService = `
---
kind: Service
apiVersion: v1
metadata:
  name: kontinuous
  namespace: {{.Namespace}}
  labels:
    app: kontinuous
    type: ci-cd
spec:
  type: LoadBalancer
  selector:
    app: kontinuous
    type: ci-cd
  ports:
    - name: api
      port: 8080
      targetPort: 3005
`

var kontinuousRC = `
---
kind: ReplicationController
apiVersion: v1
metadata:
  name: kontinuous
  namespace: {{.Namespace}}
  labels:
    app: kontinuous
    type: ci-cd
spec:
  replicas: 1
  selector:
    app: kontinuous
    type: ci-cd
  template:
    metadata:
      labels:
        app: kontinuous
        type: ci-cd
    spec:
      volumes:
        - name: kontinuous-secrets
          secret:
            secretName: kontinuous-secrets
      containers:
        - name: kontinuous
          image: quay.io/acaleph/kontinuous:latest
          imagePullPolicy: Always
          env:
            - name: KV_ADDRESS
              value: etcd:2379
            - name: S3_URL
              value: http://minio:9000
            - name: KONTINUOUS_URL
              value: http://{{.KontinuousIP}}:8080
            - name: INTERNAL_REGISTRY
              value: registry:5000
          ports:
            - name: api
              containerPort: 3005
          volumeMounts:
            - mountPath: /.secret
              name: kontinuous-secrets
              readOnly: true
`

type Deploy struct {
	Namespace    string
	AccessKey    string
	SecretKey    string
	AuthCode     string
	SecretData   string
	KontinuousIP string
}

const (
	KONTINUOUS_SPECS_FILE   = "/tmp/kontinuous-specs.yml"
	KONTINUOUS_RC_SPEC_FILE = "/tmp/kontinuous-rc-spec.yml"
)

func generateResource(templateStr string, deploy *Deploy) (string, error) {

	template := template.New("kontinuous Template")
	template, _ = template.Parse(templateStr)
	var b bytes.Buffer

	err := template.Execute(&b, deploy)

	if err != nil {
		fmt.Println(err.Error())
	}

	return b.String(), nil

}

func saveToFile(path string, data ...string) error {
	var _, err = os.Stat(path)
	var file *os.File

	if os.IsNotExist(err) {
		file, _ = os.Create(path)
		defer file.Close()
	}

	file, err = os.OpenFile(path, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	defer file.Close()
	for _, dataStr := range data {
		_, err = file.WriteString(dataStr)

		if err != nil {
			fmt.Println(err.Error())
			return err
		}
	}

	err = file.Sync()
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}

func encryptSecret(secret string) (string, error) {
	cmd := fmt.Sprintf(`echo -n "%s" | openssl base64 | tr -d "\n"`, secret)
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil

}

func createKontinuousResouces(path string) error {
	cmd := fmt.Sprintf("kubectl create -f %s", path)
	_, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}
	return nil
}

func deleteKontinuousResources(path string) error {
	cmd := fmt.Sprintf("kubectl delete -f %s", path)
	_, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}
	return nil
}

func fetchKontinuousIP(namespace string) (string, error) {
	var ip string

	cmd := fmt.Sprintf(`kubectl get svc kontinuous --namespace=%s -o template --template="{{.status.loadBalancer.ingress}}"`, namespace)
	for len(ip) == 0 {
		out, err := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			return "", err
		}

		outStr := string(out)
		if !strings.Contains(outStr, "<no value>") && !strings.Contains(outStr, "<none>") {
			ipStr := strings.TrimPrefix(outStr, "[map[ip:")
			ip = strings.TrimSuffix(ipStr, "]]")
		} else {
			time.Sleep(5 * time.Second)
		}
	}
	return ip, nil
}

func RemoveResources() error {
	err := deleteKontinuousResources(KONTINUOUS_SPECS_FILE)
	if err != nil {
		return err
	}
	err = deleteKontinuousResources(KONTINUOUS_RC_SPEC_FILE)
	if err != nil {
		return err
	}
	return nil
}

func DeployKontinuous(namespace, accesskey, secretkey, authcode string) error {
	deploy := Deploy{
		Namespace: namespace,
		AccessKey: accesskey,
		SecretKey: secretkey,
		AuthCode:  authcode,
	}
	sData, _ := generateResource(secretData, &deploy)
	encryptedSecret, _ := encryptSecret(sData)
	deploy.SecretData = encryptedSecret
	secret, _ := generateResource(secret, &deploy)
	minioStr, _ := generateResource(minio, &deploy)
	etcdStr, _ := generateResource(etcd, &deploy)
	registryStr, _ := generateResource(registry, &deploy)
	kontinuousSvcStr, _ := generateResource(kontinuousService, &deploy)

	//save string in a file
	saveToFile(KONTINUOUS_SPECS_FILE, secret, minioStr, etcdStr, registryStr, kontinuousSvcStr)
	err := createKontinuousResouces(KONTINUOUS_SPECS_FILE)

	if err != nil {
		return err
	}

	ip, _ := fetchKontinuousIP("acaleph")
	deploy.KontinuousIP = ip
	kontinuousRcStr, _ := generateResource(kontinuousRC, &deploy)
	saveToFile(KONTINUOUS_RC_SPEC_FILE, kontinuousRcStr)
	err = createKontinuousResouces(KONTINUOUS_RC_SPEC_FILE)

	if err != nil {
		return err
	}
	return nil
}
