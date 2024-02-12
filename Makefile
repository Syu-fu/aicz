include .env
run:
	@OPENAPI_KEY=$(OPENAPI_KEY) go run main.go
