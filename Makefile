# This file is generated with create-go-app: do not edit.
.PHONY: test help

# special target to export all variables
.EXPORT_ALL_VARIABLES:

## test: run linter and tests
test:
	go generate ./...
	golangci-lint run
	go test -v -count=1 ./...

help: Makefile
	@echo " Choose a command run in "$(PROJECTNAME)":"
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
