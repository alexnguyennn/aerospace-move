alias g := generate
generate:
    go generate

alias b := build
build: generate
    go build