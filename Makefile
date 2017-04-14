.PHONY: all build fmt clean test

# This builds everything except the goterm binary since that relies on external
# packages which we don't need for this project specifically
all: build bin/keyreport bin/absprompt bin/simple bin/dumb bin/prompt bin/actually_simple

SRCS = *.go

bin/keyreport: $(SRCS) example/keyreport.go
	go build -o bin/keyreport example/keyreport.go

bin/absprompt: $(SRCS) example/absprompt.go
	go build -o bin/absprompt example/absprompt.go

bin/prompt: $(SRCS) example/prompt.go
	go build -o bin/prompt example/prompt.go

bin/simple: $(SRCS) example/simple.go
	go build -o bin/simple example/simple.go

bin/dumb: $(SRCS) example/dumb.go
	go build -o bin/dumb example/dumb.go

example/actually_simple.go: terminal_test.go
	cp terminal_test.go example/actually_simple.go
	sed -i "s|package terminal_test|package main|" example/actually_simple.go
	sed -i "s|func Example|func main|" example/actually_simple.go

bin/actually_simple: $(SRCS) example/actually_simple.go
	go build -o bin/actually_simple example/actually_simple.go

bin/goterm: $(SRCS) example/goterm.go
	go build -o bin/goterm example/goterm.go

build:
	mkdir -p bin/
	go build

fmt:
	find . -name "*.go" | xargs gofmt -l -w -s

clean:
	rm bin -rf

test:
	go test .
