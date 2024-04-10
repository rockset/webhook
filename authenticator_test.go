package webhook_test

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"

	"github.com/rockset/webhook"
)

func TestNoopAuthorization(t *testing.T) {
	auth := &webhook.NoopAuthenticator{}
	err := auth.Authenticate(events.LambdaFunctionURLRequest{})
	require.NoError(t, err)
}

func TestFailAuthorization(t *testing.T) {
	auth := &webhook.FailAuthenticator{}
	err := auth.Authenticate(events.LambdaFunctionURLRequest{})
	require.Error(t, err)
}

func TestHeaderAuthorization(t *testing.T) {
	tests := map[string]struct {
		request events.LambdaFunctionURLRequest
		err     require.ErrorAssertionFunc
	}{
		"pass":      {events.LambdaFunctionURLRequest{Headers: map[string]string{"X-Secret": "secret"}}, require.NoError},
		"fail":      {events.LambdaFunctionURLRequest{Headers: map[string]string{"X-Secret": "incorrect"}}, require.Error},
		"no header": {events.LambdaFunctionURLRequest{Headers: map[string]string{}}, require.Error},
	}

	auth := &webhook.HeaderAuthenticator{
		Header: "X-Secret",
		Secret: "secret",
	}

	for name, tst := range tests {
		t.Run(name, func(t *testing.T) {
			tst.err(t, auth.Authenticate(tst.request))
		})
	}
}

func TestSignatureAuthorization(t *testing.T) {
	body := "body"
	tests := map[string]struct {
		request events.LambdaFunctionURLRequest
		err     require.ErrorAssertionFunc
	}{
		"pass":      {events.LambdaFunctionURLRequest{Headers: map[string]string{"X-Secret": "dc46983557fea127b43af721467eb9b3fde2338fe3e14f51952aa8478c13d355"}, Body: body}, require.NoError},
		"fail":      {events.LambdaFunctionURLRequest{Headers: map[string]string{"X-Secret": "incorrect"}, Body: body}, require.Error},
		"no header": {events.LambdaFunctionURLRequest{Headers: map[string]string{}, Body: body}, require.Error},
	}

	auth := &webhook.SignatureAuthenticator{
		Header:        "X-Secret",
		SigningSecret: "secret",
	}

	for name, tst := range tests {
		t.Run(name, func(t *testing.T) {
			tst.err(t, auth.Authenticate(tst.request))
		})
	}
}
