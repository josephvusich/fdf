.PHONY: install test hooks

test:
	go vet ./...
	go test -coverprofile=coverage.out $(TESTFLAGS) ./...
	go tool cover -func=coverage.out | tail -1

install: test
	go install ./...

hooks:
	cp hooks/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
