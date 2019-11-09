Pre-install steps:
```
# Create cluster
gcloud container clusters create space-agon --cluster-version=1.12 \
  --tags=game-server \
  --scopes=gke-default \
  --num-nodes=4 \
  --no-enable-autoupgrade \
  --machine-type=n1-standard-4

# Open Firewall for Agones
gcloud compute firewall-rules create gke-game-server-firewall \
  --allow tcp:7000-8000 \
  --target-tags game-server \
  --description "Firewall to allow game server tcp traffic"

# Install Agones
kubectl create namespace agones-system
kubectl apply -f https://raw.githubusercontent.com/googleforgames/agones/release-1.1.0/install/yaml/install.yaml

# Install Open Match
kubectl apply -f https://open-match.dev/install/v0.8.0-rc.1/yaml/01-open-match-core.yaml -f om-evaluator.yaml --namespace open-match

```

Commands to deploy
```

TAG=$(date +INDEV-%Y%m%d-%H%M%S) && \
REGISTRY=gcr.io/$(gcloud config list --format 'value(core.project)') && \
GOOS=js GOARCH=wasm go test github.com/googleforgames/space-agon/client/... && \
go test github.com/googleforgames/space-agon/... && \
docker build . -f Frontend.Dockerfile -t $REGISTRY/space-agon-frontend:$TAG && \
docker build . -f Dedicated.Dockerfile -t $REGISTRY/space-agon-dedicated:$TAG && \
docker build . -f Director.Dockerfile -t $REGISTRY/space-agon-director:$TAG && \
docker build . -f Mmf.Dockerfile -t $REGISTRY/space-agon-mmf:$TAG && \
docker push $REGISTRY/space-agon-frontend:$TAG && \
docker push $REGISTRY/space-agon-dedicated:$TAG && \
docker push $REGISTRY/space-agon-director:$TAG && \
docker push $REGISTRY/space-agon-mmf:$TAG && \
ESC_REGISTRY=$(echo $REGISTRY | sed -e 's/\\/\\\\/g; s/\//\\\//g; s/&/\\\&/g') && \
ESC_TAG=$(echo $TAG | sed -e 's/\\/\\\\/g; s/\//\\\//g; s/&/\\\&/g') && \
sed -E 's/image: (.*)\/([^\/]*):(.*)/image: '$ESC_REGISTRY'\/\2:'$ESC_TAG'/' deploy_template.yaml > deploy.yaml && \
kubectl apply -f deploy.yaml

```

Get External IP from:
```
kubectl get service frontend
```

Open `http://<external ip>/` in your favorite web browser.

In the web browser, open the console, and enter
```
matchmake()
```

Repeat in a second web browser window to create a second player, the players
will be connected and can play each other.


Note: Currently ships stick around after someone disconnects, and game servers
are never cleaned up.  sredig will fix this shortly.

# Additional Things to do

View Running Game Servers:
```
kubectl get gameserver
```

Delete deployment and Open Match:
```

kubectl delete psp,clusterrole,clusterrolebinding --selector=release=open-match
kubectl delete namespace open-match
kubectl delete -f deploy.yaml 

```

Clean up game servers:
```

kubectl delete gameserver --all

```

Step after getting the fleet working, but before Open Match is hooked up, can allocate single server:
```

kubectl create -f allocation.yaml -o yaml

```
Then use `connect("<ip>:<port>")` inside the javascript console.

# Game development only

Generate components file:
```
go generate github.com/googleforgames/space-agon/game/generation
```

Rebuild proto generates files:
```
docker build -f=Dockerfile.build-protos -t build-protos . && docker run --rm --mount type=bind,source="$(pwd)",target=/workdir/mount build-protos
```

Mother of all commands:
```

# Once
docker build -f=Dockerfile.build-protos -t build-protos . 

# Every update

go generate github.com/googleforgames/space-agon/game/generation && \
docker run --rm --mount type=bind,source="$(pwd)",target=/workdir/mount build-protos && \
TAG=$(date +INDEV-%Y%m%d-%H%M%S) && \
REGISTRY=gcr.io/$(gcloud config list --format 'value(core.project)') && \
GOOS=js GOARCH=wasm go test github.com/googleforgames/space-agon/client/... && \
go test github.com/googleforgames/space-agon/...





 && \
docker build . -f Frontend.Dockerfile -t $REGISTRY/space-agon-frontend:$TAG && \
docker build . -f Dedicated.Dockerfile -t $REGISTRY/space-agon-dedicated:$TAG && \
docker build . -f Director.Dockerfile -t $REGISTRY/space-agon-director:$TAG && \
docker build . -f Mmf.Dockerfile -t $REGISTRY/space-agon-mmf:$TAG && \
docker push $REGISTRY/space-agon-frontend:$TAG && \
docker push $REGISTRY/space-agon-dedicated:$TAG && \
docker push $REGISTRY/space-agon-director:$TAG && \
docker push $REGISTRY/space-agon-mmf:$TAG && \
ESC_REGISTRY=$(echo $REGISTRY | sed -e 's/\\/\\\\/g; s/\//\\\//g; s/&/\\\&/g') && \
ESC_TAG=$(echo $TAG | sed -e 's/\\/\\\\/g; s/\//\\\//g; s/&/\\\&/g') && \
sed -E 's/image: (.*)\/([^\/]*):(.*)/image: '$ESC_REGISTRY'\/\2:'$ESC_TAG'/' deploy_template.yaml > deploy.yaml && \
kubectl apply -f deploy.yaml
```
