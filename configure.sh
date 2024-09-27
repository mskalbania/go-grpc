go install google.golang.org/grpc/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
go install github.com/envoyproxy/protoc-gen-validate@latest

mkdir validate
curl -o validate/validate.proto https://raw.githubusercontent.com/bufbuild/protoc-gen-validate/refs/heads/main/validate/validate.proto

buf generate . --path proto

rm -rf validate/