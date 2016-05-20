package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"
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
var minioRc = `
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

var etcdRc = `
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

var registryRc = `
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

var kontinuousRc = `
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

var dashboardRc = `
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
      containers:
      - env:
        - name: GITHUB_CLIENT_CALLBACK
          value: http://{{.DashboardIP}}:5000
        - name: GITHUB_CLIENT_ID
          value: {{.GHClient}}
        - name: KONTINUOUS_API_URL
          value: http://{{.KontinuousIP}}:8080
        image: quay.io/acaleph/kontinuous-ui:latest
        imagePullPolicy: Always
        name: kontinuous-ui
        ports:
        - containerPort: 5000
          name: dashboard

`

type Deploy struct {
	Namespace    string
	AccessKey    string
	SecretKey    string
	AuthCode     string
	SecretData   string
	KontinuousIP string
	DashboardIP  string
	GHClient     string
	GHSecret     string
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
	cmd := fmt.Sprintf("kubectl apply -f %s", path)
	_, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return err
	}
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
	deployResource(minioRc, "MINIO_RC", &deploy)
	deployResource(etcdSvc, "ETCD_SVC", &deploy)
	deployResource(etcdRc, "ETCD_RC", &deploy)
	deployResource(registrySvc, "REGISTRY_SVC", &deploy)
	deployResource(registryRc, "REGISTRY_RC", &deploy)
	deployResource(kontinuousSvc, "KONTINUOUS_SVC", &deploy)
	deployResource(dashboardSvc, "DASHBOARD_SVC", &deploy)

	err := getS3Details(&deploy)
	if err != nil {
		fmt.Println(err.Error())
	}

	sData, _ := generateResource(secretData, &deploy)
	deploy.SecretData = encryptSecret(sData)
	deployResource(secret, "APP_SECRET", &deploy)

	ip, _ := fetchKontinuousIP("kontinuous", deploy.Namespace)
	dashboardIp, _ := fetchKontinuousIP("kontinuous-ui", deploy.Namespace)
	deploy.DashboardIP = dashboardIp
	deploy.KontinuousIP = ip

	err = deployResource(kontinuousRc, "KONTINUOUS_RC", &deploy)

	if err != nil {
		fmt.Println("Unable to deploy Kontinuous Deployment \n %s", err.Error())
		return err
	}
	err = deployResource(dashboardRc, "DASHBOARD_RC", &deploy)
	if err != nil {
		fmt.Printf("Unable to deploy Kontinuous UI Deployment \n %s", err.Error())
		return err
	}

	fmt.Println("_____________________________________________________________\n")
	fmt.Printf("Kontinuous API : http://%s:8080 \n", deploy.KontinuousIP)
	fmt.Printf("Kontinuous Dashboard : http://%s:5000 \n", deploy.DashboardIP)
	fmt.Println("_____________________________________________________________\n")

	return nil
}
