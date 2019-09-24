Commands to build:


GOOS=js GOARCH=wasm go test github.com/googleforgames/space-agon/client/... && go test github.com/googleforgames/space-agon/... && docker build . -f Frontend.Dockerfile -t space-agon-frontend && docker run -p 127.0.0.1:8080:8080/tcp space-agon-frontend
