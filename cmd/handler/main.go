package main

import (
	"os"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/rockset/webhook"
)

func main() {
	handler, err := webhook.New(os.LookupEnv)
	if err != nil {
		panic(err)
	}
	lambda.Start(handler.HandlePayload)
}
