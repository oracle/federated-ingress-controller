# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

PROJECT_NAME := federated-ingress-controller
GOFILES:=$(shell find . -name '*.go' | grep -v -E '(./vendor)')
VERSION?=$(shell git describe --tags --dirty)
IMAGE_TAG:=${DOCKER_REGISTRY}/${PROJECT_NAME}:${VERSION}

GOSRC := github.com/kubernetes-incubator/federated-ingress-controller

all: clean check bin image

image:
	docker build -t ${IMAGE_TAG} -f deploy/Dockerfile .

push_image:
	docker push ${IMAGE_TAG}

bin: bin/linux/${PROJECT_NAME}

bin/%: LDFLAGS=-X main.Version=${VERSION}
bin/%: $(GOFILES)
	mkdir -p $(dir $@)
	CGO_ENABLED=0 GOOS=$(word 1, $(subst /, ,$*)) GOARCH=amd64 go build -a -installsuffix cgo -ldflags "$(LDFLAGS)" -o "bin/linux/federatedingress-controller" ${GOSRC}/cmd/ingresscontroller/...

gofmt:
	gofmt -w -s pkg/
	gofmt -w -s cmd/

test:
	go test ${GOSRC}/pkg/... -args -v=1 -logtostderr
	go test ${GOSRC}/cmd/... -args -v=1 -logtostderr

coverage: ## Generate global code coverage report
	./tools/coverage.sh;

coverhtml: ## Generate global code coverage report in HTML
	./tools/coverage.sh html;

check:
	find . -name vendor -prune -o -name '*.go' -exec gofmt -s -d {} +
	go vet $(shell go list ./... | grep -v '/vendor/')
	go test -v $(shell go list ./... | grep -v '/vendor/')

clean:
	rm -rf bin

.PHONY: all
