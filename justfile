alias g := generate
generate:
    go generate

alias s := schema
schema:
    pkl-gen-go {{justfile_directory()}}/pkl/schema.pkl --base-path github.com/alexnguyennn/aerospace-move/

alias b := build
build: generate schema
    go build