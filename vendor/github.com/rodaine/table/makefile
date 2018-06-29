.PHONY: lint
lint:
	gofmt -d -s .
	golint -set_exit_status ./...
	go tool vet -all -shadow -shadowstrict .

.PHONY: test
test:
	go test -v -cover -race ./...
