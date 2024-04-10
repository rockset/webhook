package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log"

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

type FailAuthenticator struct{}

func (a *FailAuthenticator) Authenticate(_ events.LambdaFunctionURLRequest) error {
	return AuthFailedErr
}

type NoopAuthenticator struct{}

func (a *NoopAuthenticator) Authenticate(_ events.LambdaFunctionURLRequest) error {
	return nil
}

// SignatureAuthenticator implements SHA256 HMAC signature verification
type SignatureAuthenticator struct {
	SigningSecret string
	// Header is the name of the header that contains the signature, defaults to "X-Signature" if left empty.
	Header string
}

func (a *SignatureAuthenticator) Authenticate(req events.LambdaFunctionURLRequest) error {
	header := "X-Signature"
	if a.Header != "" {
		header = a.Header
	}

	signatureFromHeader, ok := req.Headers[header]
	if !ok {
		log.Printf("header %s not found", header)
		return AuthFailedErr
	}

	h := hmac.New(sha256.New, []byte(a.SigningSecret))
	h.Write([]byte(req.Body))
	calculatedSignature := hex.EncodeToString(h.Sum(nil))

	// Compare the calculated HMAC with the one from the header
	if hmac.Equal([]byte(calculatedSignature), []byte(signatureFromHeader)) {
		return nil
	}
	log.Printf("signature mismatch: %s != %s", calculatedSignature, signatureFromHeader)

	return AuthFailedErr
}
