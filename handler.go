package webhook

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
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
	// Paths is a map of paths to locations (workspace and collection, where workspace is optional)
	Paths Paths
}

type Paths map[string]Location

type Location struct {
	Workspace  string `json:"workspace"`
	Collection string `json:"collection"`
}

func New(env func(string) (string, error)) (*Handler, error) {
	workspace, err := env("WORKSPACE")
	if err != nil {
		return nil, err
	}

	raw, err := env("PATHS")
	if err != nil {
		return nil, err
	}
	var paths Paths
	if err := json.Unmarshal([]byte(raw), &paths); err != nil {
		return nil, err
	}

	// this reads ROCKSET_APIKEY & ROCKSET_APISERVER from the environment
	rc, err := rockset.NewClient()
	if err != nil {
		return nil, err
	}

	return &Handler{
		Rockset:   rc,
		Workspace: workspace,
		Paths:     paths,
	}, nil
}

var (
	BadDocumentErr = fmt.Errorf("failed to add document")
	MissingPathErr = fmt.Errorf("missing path configuration")
)

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

	// TODO get collection from request path
	loc, found := h.Paths[request.RawPath]
	if !found {
		return fmt.Errorf("%s: %w", request.RawPath, MissingPathErr)
	}

	var workspace string
	if loc.Workspace == "" {
		workspace = h.Workspace
	} else {
		workspace = loc.Workspace
	}

	result, err := h.Rockset.AddDocumentsRaw(ctx, workspace, loc.Collection, json.RawMessage(body))
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
