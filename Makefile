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

opentracing-example-docker:
	@echo "Build the opentracing-example docker image"
	@docker build -f deploy/Dockerfile.repo -t splunk/go-observation-repo .
	@docker build -f deploy/Dockerfile.opentracing -t splunk/opentracing-example .
	@docker images | head -1

opentracing-docker-run:
	@echo "Running opentracing-example docker containers with docker-compose. ctrl-C to stop."
	docker-compose -f deploy/docker-compose.yaml up


