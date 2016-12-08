.PHONY: all

all:
	mkdir -p bin/
	go build
	go build -o bin/keyreport example/keyreport.go
	go build -o bin/prompter example/prompter.go
	go build -o bin/simple example/simple.go

clean:
	rm bin -rf

test:
	go test .
