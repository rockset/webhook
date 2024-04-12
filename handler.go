package webhook

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/rockset/rockset-go-client"
	"github.com/rockset/rockset-go-client/openapi"
)

// go

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate -o fake . Rockset
type Rockset interface {
	AddDocumentsRaw(ctx context.Context, workspace, collection string, body json.RawMessage) ([]openapi.DocumentStatus, error)
}

type Handler struct {
	// Rockset *rockset.RockClient
	Rockset Rockset
	// Workspace is the default workspace to use if not specified in the path
	Workspace string
	// Configs is a map of endpoint configurations
	Configs Configs
	// Debug is a flag to enable debug logging
	Debug bool
}

type Configs map[string]Config

type Config struct {
	Workspace          string     `json:"workspace"`
	Collection         string     `json:"collection"`
	WrapPayloadInArray bool       `json:"wrap"`
	Auth               AuthConfig `json:"auth"`
}

type AuthConfig struct {
	// Type is the type of authentication to use, one of "header" or "signature"
	Type   string `json:"type"`
	Secret string `json:"secret"`
	Header string `json:"header"`
}

func New(env func(string) (string, bool)) (*Handler, error) {
	_, debug := env("DEBUG")

	workspace, found := env("WORKSPACE")
	if !found {
		return nil, fmt.Errorf("missing required environment variable WORKSPACE")
	}
	if debug {
		log.Printf("workspace: %s", workspace)
	}

	var raw, path string
	var err error
	raw, found = env("CONFIG")
	if !found {
		if path, found = env("CONFIG_PATH"); !found {
			return nil, fmt.Errorf("missing required environment variable CONFIG or CONFIG_PATH")
		}
		raw, err = LoadConfig(context.Background(), path)
		if err != nil {
			return nil, err
		}
	}
	if debug {
		log.Printf("raw config: %s", raw)
	}

	var configs Configs
	if err := json.Unmarshal([]byte(raw), &configs); err != nil {
		return nil, err
	}
	if debug {
		log.Printf("parsed config: %+v", configs)
	}

	// this reads ROCKSET_APIKEY & ROCKSET_APISERVER from the environment
	rc, err := rockset.NewClient()
	if err != nil {
		return nil, err
	}

	return &Handler{
		Rockset:   rc,
		Workspace: workspace,
		Configs:   configs,
		Debug:     debug,
	}, nil
}

func LoadConfig(ctx context.Context, path string) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", err
	}

	client := ssm.NewFromConfig(cfg)
	input := &ssm.GetParameterInput{
		Name:           &path,
		WithDecryption: aws.Bool(true),
	}

	resp, err := client.GetParameter(ctx, input)
	if err != nil {
		return "", err
	}
	log.Printf("loaded config from %s: v%d", path, resp.Parameter.Version)

	return *resp.Parameter.Value, nil
}

var (
	BadDocumentErr = fmt.Errorf("failed to add document")
	MissingPathErr = fmt.Errorf("missing path configuration")
)

func (h Handler) AuthenticatorForRequest(path string) Authenticator {
	cfg, found := h.Configs[path]
	if !found {
		log.Printf("no config found for %s", path)
		return &FailAuthenticator{}
	}

	switch cfg.Auth.Type {
	case "noop":
		return &NoopAuthenticator{}
	case "header":
		return &HeaderAuthenticator{
			Header: cfg.Auth.Header,
			Secret: cfg.Auth.Secret,
		}
	case "signature":
		return &SignatureAuthenticator{
			Header:        cfg.Auth.Header,
			SigningSecret: cfg.Auth.Secret,
		}
	default:
		log.Printf("unknown auth type %s", cfg.Auth.Type)
		return &FailAuthenticator{}
	}
}

func (h Handler) HandlePayload(ctx context.Context, request events.LambdaFunctionURLRequest) error {
	var body string
	if request.IsBase64Encoded {
		// decode the body
		bytes, err := base64.StdEncoding.DecodeString(request.Body)
		if err != nil {
			return err
		}
		body = string(bytes)
	} else {
		body = request.Body
	}

	if h.Debug {
		for k, v := range request.Headers {
			fmt.Printf("%s: %s\n", k, v)
		}
		fmt.Printf("Body: %s\n", body)
	}

	cfg, found := h.Configs[request.RawPath]
	if !found {
		return fmt.Errorf("%s: %w", request.RawPath, MissingPathErr)
	}

	// authenticate
	auth := h.AuthenticatorForRequest(request.RawPath)
	if err := auth.Authenticate(request); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	var workspace string
	if cfg.Workspace == "" {
		workspace = h.Workspace
	} else {
		workspace = cfg.Workspace
	}

	if cfg.WrapPayloadInArray {
		log.Printf("wrapping payload in [...]")
		body = fmt.Sprintf("[%s]", body)
	}

	result, err := h.Rockset.AddDocumentsRaw(ctx, workspace, cfg.Collection, json.RawMessage(body))
	if err != nil {
		return fmt.Errorf("failed to add documents: %w", err)
	}

	for _, doc := range result {
		if doc.GetStatus() != "ADDED" {
			return fmt.Errorf("%s.%s (%s): %w",
				workspace, doc.GetCollection(), doc.GetStatus(), BadDocumentErr)
		}
	}

	return nil
}
