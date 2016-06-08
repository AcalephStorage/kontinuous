package main

import (
	"net"
	"os"

	"encoding/json"
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/emicklei/go-restful"
	"github.com/emicklei/go-restful/swagger"
	"github.com/urfave/cli"

	"github.com/AcalephStorage/kontinuous/api"
	"github.com/AcalephStorage/kontinuous/controller"
	"github.com/AcalephStorage/kontinuous/store/kv"
)

// app name and version. this are placed in var so we can change them in runtime if needed.
var (
	appName = "kontinuous"
	version = "dev"
)

// app flag names
const (
	debugFlag         = "debug"
	bindHostFlag      = "bind-host"
	bindPortFlag      = "bind-port"
	kvAddressFlag     = "kv-address"
	kvCAFlag          = "kv-ca"
	kvCertFlag        = "kv-cert"
	kvKeyFlag         = "kv-key"
	s3urlFlag         = "s3-url"
	kubeURLFlag       = "kube-url"
	swaggerUIPathFlag = "swagger-ui-path"
	jwtSecretsFlag    = "jwt-secrets"
	s3SecretsFlag     = "s3-secrets"
	githubSecretsFlag = "github-secrets"
	kubeTokenFlag     = "kube-token"
	kubeCAFlag        = "kube-ca"
)

// exit codes
const (
	normal = iota
	loadSecretsError
	kubeClientError
	kvClientError
)

// the global app flags
var appFlags = []cli.Flag{
	cli.BoolFlag{
		Name:   debugFlag,
		EnvVar: "KONTINUOUS_DEBUG",
		Hidden: true,
		Usage:  "enable debug mode",
	},
	cli.StringFlag{
		Name:   bindHostFlag,
		EnvVar: "KONTINUOUS_BIND_HOST",
		Value:  "0.0.0.0",
		Usage:  "interface to bind the kontinuous service",
	},
	cli.StringFlag{
		Name:   bindPortFlag,
		EnvVar: "KONTINUOUS_BIND_PORT",
		Value:  "3005",
		Usage:  "port to bind the kontinuous service",
	},
	cli.StringFlag{
		Name:   kvAddressFlag,
		EnvVar: "KONTINUOUS_KV_ADDRESS",
		Value:  "localhost:2379",
		Usage:  "address of etcd",
	},
	cli.StringFlag{
		Name:   kvCAFlag,
		EnvVar: "KONTINUOUS_KV_CA",
		Value:  "",
		Usage:  "path to the CA file used by etcd",
	},
	cli.StringFlag{
		Name:   kvCertFlag,
		EnvVar: "KONTINUOUS_KV_CERT",
		Value:  "",
		Usage:  "path to the cert file used by etcd",
	},
	cli.StringFlag{
		Name:   kvKeyFlag,
		EnvVar: "KONTINUOUS_KV_KEY",
		Value:  "",
		Usage:  "path to the key file used by etcd",
	},
	cli.StringFlag{
		Name:   s3urlFlag,
		EnvVar: "KONTINUOUS_S3_URL",
		Value:  "",
		Usage:  "s3 url used for storage",
	},
	cli.StringFlag{
		Name:   kubeURLFlag,
		EnvVar: "KONTINUOUS_KUBE_URL",
		Value:  "https://kubernetes.default",
		Usage:  "the kubernetes api url",
	},
	cli.StringFlag{
		Name:   swaggerUIPathFlag,
		EnvVar: "KONTINUOUS_SWAGGER_UI_PATH",
		Value:  "/swagger",
		Usage:  "path to the swagger ui files",
	},
	cli.StringFlag{
		Name:   jwtSecretsFlag,
		EnvVar: "KONTINUOUS_JWT_SECRETS",
		Value:  "/.kontinuous/secrets/jwt/jwt.json",
		Usage:  "path to the kontinuous secrets json file",
	},
	cli.StringFlag{
		Name:   s3SecretsFlag,
		EnvVar: "KONTINUOUS_S3_SECRETS",
		Value:  "/.kontinuous/secrets/s3/s3.json",
		Usage:  "path to the s3 secrets file",
	},
	cli.StringFlag{
		Name:   githubSecretsFlag,
		EnvVar: "KONTINUOUS_GITHUB_SECRETS",
		Value:  "/.kontinuous/secrets/github/github.json",
		Usage:  "path to the github secrets file",
	},
	cli.StringFlag{
		Name:   kubeTokenFlag,
		EnvVar: "KONTINUOUS_KUBE_TOKEN",
		Value:  "/var/run/secrets/kubernetes.io/serviceaccount/token",
		Usage:  "path to the kube token file",
	},
	cli.StringFlag{
		Name:   kubeCAFlag,
		EnvVar: "KONTINUOUS_KUBE_CA",
		Value:  "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
		Usage:  "path to the kube CA file",
	},
}

// JWTSecrets is the secret required for validating JWT requests
type JWTSecrets struct {
	Secret string `json:"secret"`
}

// S3Secrets are the secrets required for accessing S3
type S3Secrets struct {
	S3AccessKey string `json:"s3AccessKey"`
	S3SecretKey string `json:"s3SecretKey"`
}

// GithubSecrets are the secrets required for working with Gihtub's OAuth
type GithubSecrets struct {
	GithubClientID     string `json:"githubClientID"`
	GithubClientSecret string `json:"githubClientSecret"`
}

var allowedCorsHeaders = []string{
	"Authorization",
	"Accept",
	"Content-Type",
	"Origin",
	"X-Custom-Event",
}

func main() {
	log.Info("Starting Kontinuous...")

	app := cli.NewApp()
	app.Name = appName
	app.Version = version
	app.Flags = appFlags
	app.Action = start
	app.Run(os.Args)
}

func start(context *cli.Context) error {
	log.Infoln("kontinuous started...")

	// enable debug mode
	debug := context.Bool(debugFlag)
	if debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("===== Debug Mode =====")
	}

	// load secrets
	var jwtSecrets JWTSecrets
	var s3Secrets S3Secrets
	var githubSecrets GithubSecrets
	secrets := map[string]interface{}{
		context.String(jwtSecretsFlag):    &jwtSecrets,
		context.String(s3SecretsFlag):     &s3Secrets,
		context.String(githubSecretsFlag): &githubSecrets,
	}
	if err := loadSecrets(secrets); err != nil {
		msg := "unable to load secrets"
		log.WithError(err).Errorln(msg)
		return cli.NewExitError(msg, loadSecretsError)
	}
	log.Infoln("secrets loaded.")

	// create restful container
	container := kontinuousRestfulContainer()

	// create backends
	cacert := context.String(kvCAFlag)
	cert := context.String(kvCertFlag)
	key := context.String(kvKeyFlag)
	address := context.String(kvAddressFlag)
	kvClient, err := createKVClient(cacert, cert, key, address)
	if err != nil {
		msg := "unable to create kv client"
		log.WithError(err).Errorln(msg)
		return cli.NewExitError(msg, kvClientError)
	}

	// create resources
	auth := createAuthResource(kvClient, jwtSecrets, githubSecrets)
	pipeline := new(api.PipelineResource)
	repo := new(api.RepositoryResource)

	// register endpoints
	auth.Register(container)
	pipeline.Register(container)
	repo.Register(container)

	// enable swagger
	swaggerUIPath := context.String(swaggerUIPathFlag)
	swaggerConfig := swagger.Config{
		WebServices: container.RegisteredWebServices(),
		ApiPath:     "/apidocs.json",
		ApiVersion:  version,
		Info: swagger.Info{
			Title:       "Kontinuous",
			Description: "Service for managing CI/CD builds through Kubernetes Jobs",
		},
		SwaggerPath:     "/apidocs/",
		SwaggerFilePath: swaggerUIPath,
	}
	swagger.RegisterSwaggerService(swaggerConfig, container)

	host := context.String(bindHostFlag)
	port := context.String(bindPortFlag)
	addr := net.JoinHostPort(host, port)

	if err := http.ListenAndServe(addr, container); err != nil {
		log.WithError(err).Errorln("Stopping kontinuous...")
	}
	log.Info("Stopping kontinuous...")
	return nil
}

func loadSecrets(secrets map[string]interface{}) error {
	for file, data := range secrets {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			log.WithError(err).Debugf("unable to read secrets file: %s", file)
			return err
		}
		if err := json.Unmarshal(content, data); err != nil {
			log.WithError(err).Debugf("unable to decode json from secrets file: %s", file)
			return err
		}
	}
	return nil
}

func kontinuousRestfulContainer() *restful.Container {
	container := restful.NewContainer()
	cors := restful.CrossOriginResourceSharing{
		AllowedHeaders: allowedCorsHeaders,
		Container:      container,
	}
	container.Filter(cors.Filter)
	container.Filter(container.OPTIONSFilter)
	return container
}

func createKVClient(cacert, cert, key, address string) (kv.Client, error) {
	kvClient, err := kv.NewEtcdClient(cacert, cert, key, address)
	if err != nil {
		log.WithError(err).Debug("unable to create etcd client")
		return nil, err
	}
	return kvClient, nil
}

func createAuthResource(kvClient kv.Client, jwtSecrets JWTSecrets, githubSecrets GithubSecrets) *api.AuthResource {
	// datastore
	userStore := &kv.UserStore{KVClient: kvClient}
	userMapStore := &kv.UserMapStore{KVClient: kvClient}

	// controller
	userController := &controller.UserController{
		UserStore:    userStore,
		UserMapStore: userMapStore,
	}
	controller := &controller.AuthController{
		UserController:     userController,
		JWTSecret:          jwtSecrets.Secret,
		GithubClientID:     githubSecrets.GithubClientID,
		GithubClientSecret: githubSecrets.GithubClientSecret,
	}
	authFilter := &api.AuthFilter{AuthController: controller}

	// resource
	authResource := &api.AuthResource{AuthController: controller, AuthFilter: authFilter}
	return authResource
}
