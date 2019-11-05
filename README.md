Commands to build:


GOOS=js GOARCH=wasm go test github.com/googleforgames/space-agon/client/... && \
go test github.com/googleforgames/space-agon/... && \
docker build . -f Frontend.Dockerfile -t space-agon-frontend && \
docker run -p 127.0.0.1:8080:8080/tcp space-agon-frontend

Pre-instal steps:
```
kubectl apply -f https://open-match.dev/install/v0.8.0-rc.1/yaml/01-open-match-core.yaml -f om-evaluator.yaml --namespace open-match

kubectl apply -f https://open-match.dev/install/v0.8.0-rc.1/yaml/install.yaml -f om-evaluator.yaml --namespace open-match

kubectl apply -f https://open-match.dev/install/v0.8.0-rc.1/yaml/03-prometheus-chart.yaml -f https://open-match.dev/install/v0.8.0-rc.1/yaml/04-grafana-chart.yaml -f https://open-match.dev/install/v0.8.0-rc.1/yaml/05-jaeger-chart.yaml --namespace open-match


kubectl apply -f https://open-match.dev/install/v0.8.0-rc.1/yaml/01-open-match-core.yaml -f om-evaluator.yaml --namespace open-match


```



Commands to deploy
```

TAG=$(date +INDEV-%Y%m%d-%H%M%S) && \
REGISTRY=gcr.io/$(gcloud config list --format 'value(core.project)')
GOOS=js GOARCH=wasm go test github.com/googleforgames/space-agon/client/... && \
go test github.com/googleforgames/space-agon/... && \
docker build . -f Frontend.Dockerfile -t $REGISTRY/space-agon-frontend:$TAG && \
docker build . -f Dedicated.Dockerfile -t $REGISTRY/space-agon-dedicated:$TAG && \
docker build . -f Director.Dockerfile -t $REGISTRY/space-agon-director:$TAG && \
docker build . -f Mmf.Dockerfile -t $REGISTRY/space-agon-mmf:$TAG && \
docker push $REGISTRY/space-agon-frontend && \
docker push $REGISTRY/space-agon-dedicated && \
docker push $REGISTRY/space-agon-director && \
docker push $REGISTRY/space-agon-mmf && \
sed -E -i 's/image: (.*):(.*)/image: \1:'$TAG'/' deploy.yaml && \
kubectl apply -f deploy.yaml






```



Delete everything:
```

kubectl delete psp,clusterrole,clusterrolebinding --selector=release=open-match
kubectl delete namespace open-match
kubectl delete -f deploy.yaml 

```


kubectl delete gameserver --all


Various commands before hooking up the director.
```
kubectl create -f gameserver.yaml
kubectl create -f allocation.yaml -o yaml
kubectl describe gameserver
```
