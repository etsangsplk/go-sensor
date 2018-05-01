.PHONY := help lint test coverage

help:   ## Show this help, automatically generated from comments in the Makefile
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/:.*##/         /'

lint:  ## Runs lint tests against the project
	gometalinter --exclude=vendor

units:  ## Runs go unit tests
	go test -v ./...

coverage: ## Generates test coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
