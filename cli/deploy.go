package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"encoding/base64"
	"encoding/pem"

	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
)

var namespaceData = `
---
kind: Namespace
apiVersion: v1
metadata:
  name: {{.Namespace}}
`

var secretData = `
{
  "AuthSecret": "{{.AuthCode}}",
  "S3SecretKey": "{{.SecretKey}}",
  "S3AccessKey": "{{.AccessKey}}",
  "GithubClientID": "{{.GHClient}}",
  "GithubClientSecret": "{{.GHSecret}}"
}

`

var sslSecrets = `
---
apiVersion: v1
kind: Secret
metadata:
  name: ssl-secret
  namespace: {{.Namespace}}
data:
  cert: {{.Cert}}
  key: {{.Key}}
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

var minioSvc = `
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
`
var minioDep = `
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: minio
  namespace: {{.Namespace}}
  labels:
    app: minio
    type: object-store
spec:
  replicas: 1
  selector:
    matchLabels:
      app: minio
      type: object-store
  template:
    metadata:
      name: minio
      labels:
        app: minio
        type: object-store
    spec:
      volumes:
        - name: empty-dir
          emptyDir: {}
      containers:
        - name: minio
          image: minio/minio:latest
          imagePullPolicy: Always
          args:
            - /data
          volumeMounts:
            - name: empty-dir
              mountPath: /data
          ports:
            - name: service
              containerPort: 9000
          livenessProbe:
            tcpSocket:
              port: 9000
            timeoutSeconds: 1
`

var etcdSvc = `
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
`

var etcdDep = `
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: etcd
  namespace: {{.Namespace}}
  labels:
    app: etcd
    type: kv
spec:
  replicas: 1
  selector:
    matchLabels:
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

var registrySvc = `
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
`

var registryDep = `
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: registry
  namespace: {{.Namespace}}
  labels:
    app: registry
    type: storage
spec:
  replicas: 1
  selector:
    matchLabels:
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

var kontinuousSvc = `
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

var kontinuousDep = `
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: kontinuous
  namespace: {{.Namespace}}
  labels:
    app: kontinuous
    type: ci-cd
spec:
  replicas: 1
  selector:
    matchLabels:
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
        - name: ssl
          secret:
            secretName: ssl-secret
      containers:
        - name: kontinuous
          image: quay.io/acaleph/kontinuous:latest
          imagePullPolicy: Always
          env:
            - name: KV_ADDRESS
              value: etcd:2379
            - name: S3_URL
              value: http://minio.{{.Namespace}}:9000
            - name: KONTINUOUS_URL
              value: https://{{.KontinuousIP}}:8080
            - name: INTERNAL_REGISTRY
              value: {{.RegistryIP}}:5000
          ports:
            - name: api
              containerPort: 3005
          volumeMounts:
            - mountPath: /.secret
              name: kontinuous-secrets
              readOnly: true
            - mountPath: /.certs
              name: ssl
              readOnly: true
`

var dashboardSvc = `
---
apiVersion: v1
kind: Service
metadata:
  labels:
    service: kontinuous-ui
    type: dashboard
  name: kontinuous-ui
  namespace: {{.Namespace}}
spec:
  ports:
  - name: dashboard
    port: 5000
    protocol: TCP
    targetPort: 5000
  selector:
    app: kontinuous-ui
    type: dashboard
  type: LoadBalancer

`

var dashboardDep = `
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    app: kontinuous-ui
    type: dashboard
  name: kontinuous-ui
  namespace: {{.Namespace}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kontinuous-ui
      type: dashboard
  template:
    metadata:
      labels:
        app: kontinuous-ui
        type: dashboard
      name: kontinuous-ui
      namespace: {{.Namespace}}
    spec:
      volumes:
        - name: ssl
          secret:
            secretName: ssl-secret
      containers:
      - env:
        - name: GITHUB_CLIENT_CALLBACK
          value: https://{{.DashboardIP}}:5000
        - name: GITHUB_CLIENT_ID
          value: {{.GHClient}}
        - name: KONTINUOUS_API_URL
          value: https://{{.KontinuousIP}}:8080
        image: quay.io/acaleph/kontinuous-ui:latest
        imagePullPolicy: Always
        name: kontinuous-ui
        ports:
        - containerPort: 5000
          name: dashboard
        volumeMounts:
        - name: ssl
          readOnly: true
          mountPath: /secrets/ssl

`

type Deploy struct {
	Namespace    string
	AccessKey    string
	SecretKey    string
	AuthCode     string
	SecretData   string
	KontinuousIP string
	DashboardIP  string
	RegistryIP   string
	GHClient     string
	GHSecret     string
	Cert         string
	Key          string
}

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

	file, err = os.OpenFile(path, os.O_RDWR, 0644)
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

func encryptSecret(secret string) string {
	return base64.StdEncoding.EncodeToString([]byte(secret))
}

func createKontinuousResouces(path string) error {
	// cmd := fmt.Sprintf("kubectl apply -f %s", path)
	// _, err := exec.Command("bash", "-c", cmd).Output()
	// if err != nil {
	// 	return err
	// }
	return nil
}

func deleteKontinuousResources() error {
	//remove namespace file
	fmt.Println("Removing Kontinuous resources ... ")
	cmd := fmt.Sprintf("rm -f /tmp/deploy/APP_NAMESPACE.yml")
	_, err := exec.Command("bash", "-c", cmd).Output()

	if err != nil {
		fmt.Println("Unable to remove namespace resource")
	}
	cmd = fmt.Sprintf("kubectl delete -f %s", "/tmp/deploy/")
	result, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}
	fmt.Printf("%s", string(result))
	return nil
}

func fetchKontinuousIP(serviceName, namespace string) (string, error) {
	var ip string

	// TODO: test out {{range .status.loadBalancer.ingress }}{{.ip}}{{end}}
	cmd := fmt.Sprintf(`kubectl get svc %s --namespace=%s -o template --template="{{.status.loadBalancer.ingress}}"`, serviceName, namespace)
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

func fetchClusterIP(serviceName, namespace string) (string, error) {
	var ip string

	// TODO: test out {{range .status.loadBalancer.ingress }}{{.ip}}{{end}}
	cmd := fmt.Sprintf(`kubectl get svc %s --namespace=%s -o template --template="{{.spec.clusterIP}}"`, serviceName, namespace)
	for len(ip) == 0 {
		out, err := exec.Command("bash", "-c", cmd).Output()
		if err != nil {
			return "", err
		}

		outStr := string(out)
		if !strings.Contains(outStr, "<no value>") && !strings.Contains(outStr, "<none>") {
			ip = strings.TrimSpace(outStr)
		} else {
			time.Sleep(5 * time.Second)
		}
	}
	return ip, nil
}

func deployResource(definition string, fileName string, deploy *Deploy) error {
	resource, _ := generateResource(definition, deploy)
	resourceArr := strings.Split(fileName, "_")
	resourceName := resourceArr[0]
	resourceType := resourceArr[1]

	cmd := fmt.Sprintf("mkdir -p %s", "/tmp/deploy")
	_, _ = exec.Command("bash", "-c", cmd).Output()

	filePath := fmt.Sprintf("/tmp/deploy/%s.yml", fileName)
	saveToFile(filePath, resource)
	//check if it exist
	cmd = fmt.Sprintf("kubectl get -f %s", filePath)
	_, err := exec.Command("bash", "-c", cmd).Output()
	if err == nil {
		fmt.Sprintf("%s: %s already exists, applying resource again... \n", resourceType, resourceName)
	}

	err = createKontinuousResouces(filePath)
	if err != nil {
		fmt.Sprintf("Unable to deploy resource: %s \n", err)
		return err
	}
	fmt.Printf("Successfully deployed %s - %s \n", resourceName, resourceType)
	return nil
}

func getS3Details(deploy *Deploy) error {

	var podName string
	cmd := fmt.Sprintf(`kubectl get po --namespace=%s -l app=minio,type=object-store --no-headers | awk '{print $1}'`, deploy.Namespace)
	waitingTime := 0

	for len(podName) == 0 && waitingTime < 30 {
		pod, _ := exec.Command("bash", "-c", cmd).Output()
		podName = string(pod)
		time.Sleep(2 * time.Second)
		waitingTime += 2
	}

	if len(podName) == 0 {
		return errors.New("Unable to Deploy Kontinuous. Dependency Minio Storage is unavailable")
	}

	podName = strings.TrimSpace(podName)
	cmd = fmt.Sprintf(`kubectl logs --namespace=%s %s | grep AccessKey | awk '{print $2}'`, deploy.Namespace, podName)
	s3AccessKey, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		fmt.Println(err.Error())
		return errors.New("Unable to Deploy Kontinuous. Dependency Minio Storage Access Key is unavailable")
	}

	deploy.AccessKey = strings.TrimSpace(string(s3AccessKey))
	cmd = fmt.Sprintf(`kubectl logs --namespace=%s %s | grep AccessKey | awk '{print $4}'`, deploy.Namespace, podName)
	s3AccessSecret, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return errors.New("Unable to Deploy Kontinuous. Dependency Minio Storage Secret Key is unavailable")
	}

	deploy.SecretKey = strings.TrimSpace(string(s3AccessSecret))
	return nil
}

func RemoveResources() error {
	err := deleteKontinuousResources()
	if err != nil {
		fmt.Printf("Unable to remove Kontinuous resources.  %s \n", err.Error())
		return err
	}
	return nil
}

func generateCertandKey() (string, string, error) {
	random := rand.Reader
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		fmt.Println(err)
		return "", "", err
	}

	now := time.Now()
	then := now.Add(60 * 60 * 24 * 365 * 1000 * 1000 * 1000)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "kontinuous",
			Organization: []string{"kontinuous"},
		},
		NotBefore:             now,
		NotAfter:              then,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		BasicConstraintsValid: true,
		IsCA: true,
	}

	derBytes, err := x509.CreateCertificate(random, &template,
		&template, &key.PublicKey, key)

	if err != nil {
		fmt.Println("Unable to create certificate")
		return "", "", err
	}

	var certBytes, keyBytes bytes.Buffer

	certPEMFile := bufio.NewWriter(&certBytes)
	pem.Encode(certPEMFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	certPEMFile.Flush()
	certPEMb64 := base64.StdEncoding.EncodeToString(certBytes.Bytes())

	keyPEMFile := bufio.NewWriter(&keyBytes)
	pem.Encode(keyPEMFile, &pem.Block{Type: "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key)})
	keyPEMFile.Flush()
	keyPEMb64 := base64.StdEncoding.EncodeToString(keyBytes.Bytes())

	return certPEMb64, keyPEMb64, nil

}

func DeployKontinuous(namespace, authcode, clientid, clientsecret string) error {
	fmt.Println("Deploying Kontinuous resources ...")
	deploy := Deploy{
		Namespace: namespace,
		AuthCode:  authcode,
		GHClient:  clientid,
		GHSecret:  clientsecret,
	}

	deployResource(namespaceData, "APP_NAMESPACE", &deploy)
	deployResource(minioSvc, "MINIO_SVC", &deploy)
	deployResource(minioDep, "MINIO_DEPLOYMENT", &deploy)
	deployResource(etcdSvc, "ETCD_SVC", &deploy)
	deployResource(etcdDep, "ETCD_DEPLOYMENT", &deploy)
	deployResource(registrySvc, "REGISTRY_SVC", &deploy)
	deployResource(registryDep, "REGISTRY_DEPLOYMENT", &deploy)
	deployResource(kontinuousSvc, "KONTINUOUS_SVC", &deploy)
	deployResource(dashboardSvc, "DASHBOARD_SVC", &deploy)

	err := getS3Details(&deploy)
	if err != nil {
		fmt.Println(err.Error())
	}

	sData, _ := generateResource(secretData, &deploy)
	deploy.SecretData = encryptSecret(sData)
	deployResource(secret, "APP_SECRET", &deploy)

	deploy.Cert, deploy.Key, _ = generateCertandKey()
	deploy.KontinuousIP, _ = fetchKontinuousIP("kontinuous", deploy.Namespace)
	deploy.DashboardIP, _ = fetchKontinuousIP("kontinuous-ui", deploy.Namespace)
	deploy.RegistryIP, _ = fetchClusterIP("registry", deploy.Namespace)

	deployResource(sslSecrets, "SSL_SECRET", &deploy)

	err = deployResource(kontinuousDep, "KONTINUOUS_DEPLOYMENT", &deploy)

	if err != nil {
		fmt.Println("Unable to deploy Kontinuous Api \n %s", err.Error())
		return err
	}
	err = deployResource(dashboardDep, "DASHBOARD_DEPLOYMENT", &deploy)
	if err != nil {
		fmt.Printf("Unable to deploy Kontinuous UI  \n %s", err.Error())
		return err
	}

	fmt.Println("_____________________________________________________________\n")
	fmt.Printf("Kontinuous API : https://%s:8080 \n", deploy.KontinuousIP)
	fmt.Printf("Kontinuous Dashboard : https://%s:5000 \n", deploy.DashboardIP)
	fmt.Println("_____________________________________________________________\n")

	return nil
}
