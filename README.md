Commands to build:


GOOS=js GOARCH=wasm go test github.com/googleforgames/space-agon/client/... && \
go test github.com/googleforgames/space-agon/... && \
docker build . -f Frontend.Dockerfile -t space-agon-frontend && \
docker run -p 127.0.0.1:8080:8080/tcp space-agon-frontend


Commands to deploy
```

TAG=v4
GOOS=js GOARCH=wasm go test github.com/googleforgames/space-agon/client/... && \
go test github.com/googleforgames/space-agon/... && \
docker build . -f Frontend.Dockerfile -t gcr.io/$(gcloud config list --format 'value(core.project)')/space-agon-frontend:$TAG && \
docker build . -f Dedicated.Dockerfile -t gcr.io/$(gcloud config list --format 'value(core.project)')/space-agon-dedicated:$TAG && \
docker push gcr.io/$(gcloud config list --format 'value(core.project)')/space-agon-frontend && \
docker push gcr.io/$(gcloud config list --format 'value(core.project)')/space-agon-dedicated && \
kubectl apply -f deploy.yaml && \
kubectl create -f gameserver.yaml
```

kubectl delete -f deploy.yaml 

&& \




kubectl delete gameserver --all



  kubectl describe gameserver
