REGISTRY:=docker.io
GROUP:=panjjo
PROJECT:=sipserver
TAG:=1.0.6

all: build docker push

build:
	GOOS=linux go build -v -o srv

docker:
	docker image rm -f ${GROUP}/${PROJECT}:${TAG}
	docker build -t ${GROUP}/${PROJECT}:${TAG} .

push:
	docker tag ${GROUP}/${PROJECT}:${TAG} ${REGISTRY}/${GROUP}/${PROJECT}:${TAG}
	docker push ${REGISTRY}/${GROUP}/${PROJECT}:${TAG}

