package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"github.com/aws/aws-lambda-go/events"
)

type Authenticator interface {
	Authenticate(req events.LambdaFunctionURLRequest) error
}

var AuthFailedErr = errors.New("authentication failed")

type HeaderAuthenticator struct {
	Header string
	Secret string
}

func (a *HeaderAuthenticator) Authenticate(req events.LambdaFunctionURLRequest) error {
	if secret, ok := req.Headers[a.Header]; ok {
		if secret == a.Secret {
			return nil
		}
	}

	return AuthFailedErr
}

type NoopAuthenticator struct{}

func (a *NoopAuthenticator) Authenticate(_ events.LambdaFunctionURLRequest) error {
	return nil
}

type SignatureAuthenticator struct {
	SigningSecret string
}

func (a *SignatureAuthenticator) Authenticate(req events.LambdaFunctionURLRequest) error {
	signatureFromHeader, ok := req.Headers["X-Signature"]
	if !ok {
		return AuthFailedErr
	}

	h := hmac.New(sha256.New, []byte(a.SigningSecret))
	h.Write([]byte(req.Body))
	calculatedSignature := hex.EncodeToString(h.Sum(nil))

	// Compare the calculated HMAC with the one from the header
	if hmac.Equal([]byte(calculatedSignature), []byte(signatureFromHeader)) {
		return nil
	}

	return AuthFailedErr
}
