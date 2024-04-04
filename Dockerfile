
FROM golang:1 as build
WORKDIR /build

# Copy dependencies list
COPY go.mod go.sum ./

# Build with optional lambda.norpc tag
COPY . .
RUN go build -tags lambda.norpc -o /handler cmd/handler/main.go

# Copy artifacts to a clean image
FROM public.ecr.aws/lambda/provided:al2023
COPY --from=build /handler /handler
ENTRYPOINT [ "/handler" ]
