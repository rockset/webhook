package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/rockset/webhook"
)

func main() {
	handler, err := webhook.New(requiredEnv)
	if err != nil {
		panic(err)
	}
	lambda.Start(handler.HandlePayload)
}

func requiredEnv(key string) (string, error) {
	val, found := os.LookupEnv(key)
	if !found {
		return "", fmt.Errorf("missing required environment variable %s", key)
	}

	return val, nil
}
