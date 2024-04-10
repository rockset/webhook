package webhook_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/rockset/rockset-go-client"
	"github.com/rockset/rockset-go-client/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rockset/webhook"
	"github.com/rockset/webhook/fake"
)

type TestCase struct {
	defaultWorkspace   string
	config             webhook.Configs
	request            events.LambdaFunctionURLRequest
	response           []openapi.DocumentStatus
	expectedWorkspace  string
	expectedCollection string
	err                error
	wantErr            error
}

var (
	testError = errors.New("test error")
	jsonDoc   = `{"key": "value"}`
)

var tests = map[string]TestCase{
	"pass": {
		expectedWorkspace:  "workspace",
		expectedCollection: "collection",
		defaultWorkspace:   "commons",
		config: webhook.Configs{
			"/path": {
				Workspace:  "workspace",
				Collection: "collection",
				Auth:       webhook.AuthConfig{Type: "noop"},
			},
		},
		request: events.LambdaFunctionURLRequest{
			RawPath: "/path",
			Body:    jsonDoc,
		},
		response: []openapi.DocumentStatus{
			{
				Collection: openapi.PtrString("collection"),
				Status:     openapi.PtrString("ADDED"),
			},
		},
	},
	"bad document status": {
		expectedWorkspace:  "workspace",
		expectedCollection: "collection",
		defaultWorkspace:   "workspace",
		config: webhook.Configs{
			"/path": {
				Collection: "collection",
				Auth:       webhook.AuthConfig{Type: "noop"},
			},
		},
		request: events.LambdaFunctionURLRequest{
			RawPath: "/path",
			Body:    jsonDoc,
		},
		response: []openapi.DocumentStatus{
			{
				Collection: openapi.PtrString("collection"),
				Status:     openapi.PtrString("FAILED"),
			},
		},
		wantErr: webhook.BadDocumentErr,
	},
	"bad api call": {
		expectedWorkspace:  "workspace",
		expectedCollection: "collection",
		defaultWorkspace:   "workspace",
		config: webhook.Configs{
			"/path": {
				Workspace:  "workspace",
				Collection: "collection",
				Auth:       webhook.AuthConfig{Type: "noop"},
			},
		},
		request: events.LambdaFunctionURLRequest{
			RawPath:         "/path",
			Body:            base64.StdEncoding.EncodeToString([]byte(jsonDoc)),
			IsBase64Encoded: true,
		},
		err:     testError,
		wantErr: testError,
	},
	"bad path": {
		expectedWorkspace:  "workspace",
		expectedCollection: "collection",
		defaultWorkspace:   "workspace",
		config: webhook.Configs{
			"/path": {
				Workspace:  "workspace",
				Collection: "collection",
				Auth:       webhook.AuthConfig{Type: "noop"},
			},
		},
		request: events.LambdaFunctionURLRequest{
			RawPath: "/missing",
			Body:    jsonDoc,
		},
		err:     testError,
		wantErr: webhook.MissingPathErr,
	},
}

func TestHandler_HandlePayload(t *testing.T) {
	t.Parallel()

	for name, tst := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			rc := &fake.FakeRockset{}
			rc.AddDocumentsRawStub = addDocumentsStub(t, tst)

			h := webhook.Handler{
				Rockset:   rc,
				Workspace: tst.defaultWorkspace,
				Configs:   tst.config,
			}

			err := h.HandlePayload(ctx, tst.request)
			if tst.wantErr != nil {
				require.ErrorIs(t, err, tst.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFoo(t *testing.T) {
	x := `{"/path": {"workspace": "workspace", "collection": "collection"}}`
	var paths webhook.Configs
	err := json.Unmarshal([]byte(x), &paths)
	assert.NoError(t, err)
	assert.Contains(t, paths, "/path")
	// assert.Equal(t, paths["/path"], webhook.Path{"workspace": "workspace", "collection": "collection"})
}

func fakeEnv(tst TestCase) func(name string) (string, error) {
	return func(name string) (string, error) {
		switch name {
		case "WORKSPACE":
			return tst.defaultWorkspace, nil
		case "PATHS":
			raw, err := json.Marshal(tst.config)
			if err != nil {
				return "", err
			}
			return string(raw), nil
		default:
			return "", fmt.Errorf("missing env var %s", name)
		}
	}
}

func addDocumentsStub(t *testing.T, tst TestCase) func(ctx context.Context, workspace, collection string,
	body json.RawMessage) ([]openapi.DocumentStatus, error) {
	return func(ctx context.Context, workspace, collection string, body json.RawMessage) ([]openapi.DocumentStatus, error) {
		assert.Equal(t, tst.expectedWorkspace, workspace)
		assert.Equal(t, tst.expectedCollection, collection)
		if tst.request.IsBase64Encoded {
			assert.JSONEq(t, jsonDoc, string(body))
		} else {
			assert.JSONEq(t, tst.request.Body, string(body))
		}

		return tst.response, tst.err
	}
}

func TestHandler_HandlePayload_integration(t *testing.T) {
	t.Skip("skipping integration test")
	t.Parallel()

	rc, err := rockset.NewClient()
	require.NoError(t, err)

	for name, tst := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			h := webhook.Handler{
				Rockset:   rc,
				Workspace: tst.defaultWorkspace,
				Configs:   tst.config,
			}
			err := h.HandlePayload(ctx, tst.request)
			if tst.wantErr != nil {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
