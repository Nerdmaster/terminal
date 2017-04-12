.PHONY: all build fmt clean test

# This builds everything except the goterm binary since that relies on external
# packages which we don't need for this project specifically
all: build bin/keyreport bin/absprompt bin/simple bin/dumb bin/prompt

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
