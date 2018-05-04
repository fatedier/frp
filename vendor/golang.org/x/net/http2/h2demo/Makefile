# Copyright 2018 The Go Authors. All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

MUTABLE_VERSION ?= latest
VERSION ?= $(shell git rev-parse --short HEAD)

IMAGE_STAGING := gcr.io/go-dashboard-dev/h2demo
IMAGE_PROD := gcr.io/symbolic-datum-552/h2demo

DOCKER_IMAGE_build0=build0/h2demo:latest
DOCKER_CTR_build0=h2demo-build0

build0: *.go Dockerfile.0
	docker build --force-rm -f Dockerfile.0 --tag=$(DOCKER_IMAGE_build0) ../..

h2demo: build0
	docker create --name $(DOCKER_CTR_build0) $(DOCKER_IMAGE_build0)
	docker cp $(DOCKER_CTR_build0):/go/bin/$@ $@
	docker rm $(DOCKER_CTR_build0)

ca-certificates.crt:
	docker create --name $(DOCKER_CTR_build0) $(DOCKER_IMAGE_build0)
	docker cp $(DOCKER_CTR_build0):/etc/ssl/certs/$@ $@
	docker rm $(DOCKER_CTR_build0)

update-deps:
	go install golang.org/x/build/cmd/gitlock
	gitlock --update=Dockerfile.0 --ignore=golang.org/x/net --tags=h2demo golang.org/x/net/http2/h2demo

docker-prod: Dockerfile h2demo ca-certificates.crt
	docker build --force-rm --tag=$(IMAGE_PROD):$(VERSION) .
	docker tag $(IMAGE_PROD):$(VERSION) $(IMAGE_PROD):$(MUTABLE_VERSION)
docker-staging: Dockerfile h2demo ca-certificates.crt
	docker build --force-rm --tag=$(IMAGE_STAGING):$(VERSION) .
	docker tag $(IMAGE_STAGING):$(VERSION) $(IMAGE_STAGING):$(MUTABLE_VERSION)

push-prod: docker-prod
	gcloud docker -- push $(IMAGE_PROD):$(MUTABLE_VERSION)
	gcloud docker -- push $(IMAGE_PROD):$(VERSION)
push-staging: docker-staging
	gcloud docker -- push $(IMAGE_STAGING):$(MUTABLE_VERSION)
	gcloud docker -- push $(IMAGE_STAGING):$(VERSION)

deploy-prod: push-prod
	kubectl set image deployment/h2demo-deployment h2demo=$(IMAGE_PROD):$(VERSION)
deploy-staging: push-staging
	kubectl set image deployment/h2demo-deployment h2demo=$(IMAGE_STAGING):$(VERSION)

.PHONY: clean
clean:
	$(RM) h2demo
	$(RM) ca-certificates.crt

FORCE:
