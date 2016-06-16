package main

import (
	"net"
	"os"

	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/swagger"
	"golang.org/x/net/http2"

	"github.com/AcalephStorage/kontinuous/api"
	"github.com/AcalephStorage/kontinuous/kube"
	"github.com/AcalephStorage/kontinuous/store/kv"
	"github.com/AcalephStorage/kontinuous/store/mc"
	"github.com/AcalephStorage/kontinuous/util"
)

const (
	SecretFile = "/.secret/kontinuous-secrets"
	Version    = "0.0.1"
)

type Secrets struct {
	AuthSecret         string
	S3SecretKey        string
	S3AccessKey        string
	GithubClientID     string
	GithubClientSecret string
}

var (
	mainLog            = util.NewContextLogger("main")
	allowedCorsHeaders = []string{
		"Authorization",
		"Accept",
		"Content-Type",
		"Origin",
		"X-Custom-Event",
	}
)

func main() {

	// log settings
	setLogLevel()
	log := mainLog.InFunc("main")

	log.Info("Starting Kontinuous...")

	// set environment variables
	setEnv()
	// params.. TODO: should be flags & env vars
	bindAddr := getEnv("BIND_ADDR", "0.0.0.0")
	bindPort := getEnv("BIND_PORT", "3005")
	kvAddress := getEnv("KV_ADDRESS", "localhost:2379")
	s3Url := getEnv("S3_URL", "")
	s3Access := getEnv("S3_ACCESS_KEY", "")
	s3Secret := getEnv("S3_SECRET_KEY", "")

	container := createRestfulContainer()

	kvClient := createKVClient(kvAddress)
	kubeClient, err := kube.NewClient("https://kubernetes.default")
	if err != nil {
		log.WithError(err).Fatal("unable to create kubernetes client")
	}

	auth := &api.AuthResource{KVClient: kvClient}
	pipeline := &api.PipelineResource{
		KVClient:    kvClient,
		MinioClient: createMinioClient(s3Url, s3Access, s3Secret),
		KubeClient:  kubeClient,
	}
	repos := &api.RepositoryResource{}

	auth.Register(container)
	pipeline.Register(container)
	repos.Register(container)

	swaggerUIPath := getEnv("SWAGGER_UI", "")
	swaggerConfig := swagger.Config{
		WebServices: container.RegisteredWebServices(),
		ApiPath:     "/apidocs.json",
		ApiVersion:  Version,
		Info: swagger.Info{
			Title:       "Kontinuous",
			Description: "Service for managing CI/CD builds through Kubernetes Jobs",
		},
		SwaggerPath:     "/apidocs/",
		SwaggerFilePath: swaggerUIPath,
	}
	swagger.RegisterSwaggerService(swaggerConfig, container)

	addr := net.JoinHostPort(bindAddr, bindPort)
	server := &http.Server{Addr: addr, Handler: container}

	if err := http2.ConfigureServer(server, nil); err != nil {
		log.WithError(err).Errorln("unable to configure http2")
		os.Exit(1)
	}

	log.Infof("Listening on: %s", addr)

	certFile := getEnv("CERT_FILE", "/.certs/cert")
	keyFile := getEnv("KEY_FILE", "/.certs/key")
	log.Fatal(server.ListenAndServeTLS(certFile, keyFile))
}

func setLogLevel() {
	logrus.SetLevel(logrus.InfoLevel)
	debug := getEnv("DEBUG", "false")
	if debug == "true" {
		logrus.SetLevel(logrus.DebugLevel)
	}
}

func getEnv(key, defaultStr string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultStr
	}
	mainLog.InFunc("getEnv").Debugf("Parameter: %s=%s", key, value)
	return value
}

func createRestfulContainer() *restful.Container {
	container := restful.NewContainer()
	cors := restful.CrossOriginResourceSharing{
		AllowedHeaders: allowedCorsHeaders,
		Container:      container,
	}
	container.Filter(cors.Filter)
	container.Filter(container.OPTIONSFilter)
	return container
}

func createKVClient(address string) kv.KVClient {
	kvClient, err := kv.NewEtcdClient("", "", "", address)
	if err != nil {
		mainLog.InFunc("createKvClient").
			WithError(err).
			Fatal("unable to create kv client")
		os.Exit(1)
	}
	return kvClient
}

func createMinioClient(url, access, secret string) *mc.MinioClient {
	minioClient, err := mc.NewMinioClient(url, access, secret)
	if err != nil {
		mainLog.InFunc("createMinioClient").
			WithError(err).
			Fatal("unable to create mc client")
		os.Exit(1)
	}
	return minioClient
}

func getSecrets() *Secrets {
	content, err := ioutil.ReadFile(SecretFile)
	if err != nil {
		mainLog.InFunc("getSecrets").
			WithError(err).
			Fatalf("Unable to read file: %s", SecretFile)
		os.Exit(1)
	}

	var secrets Secrets
	err = json.Unmarshal(content, &secrets)
	if err != nil {
		mainLog.InFunc("getSecrets").
			WithError(err).
			Fatalf("Unable to parse data from %s", SecretFile)
		os.Exit(1)
	}
	return &secrets
}

func setEnv() {
	if secrets := getSecrets(); secrets != nil {
		os.Setenv("AUTH_SECRET", secrets.AuthSecret)
		os.Setenv("S3_ACCESS_KEY", secrets.S3AccessKey)
		os.Setenv("S3_SECRET_KEY", secrets.S3SecretKey)
		os.Setenv("GITHUB_CLIENT_ID", secrets.GithubClientID)
		os.Setenv("GITHUB_CLIENT_SECRET", secrets.GithubClientSecret)
	}
}
