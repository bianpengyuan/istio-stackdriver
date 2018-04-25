SHELL := /bin/bash

.PHONY: docker

-include .rc.mk

LOCAL_OS := $(shell uname)
ifeq ($(LOCAL_OS),Linux)
   export GOOS_LOCAL = linux
else ifeq ($(LOCAL_OS),Darwin)
   export GOOS_LOCAL = darwin
else
   $(error "This system's OS $(LOCAL_OS) isn't recognized/supported")
   # export GOOS_LOCAL ?= windows
endif

export GOOS ?= $(GOOS_LOCAL)

BUILDTYPE_DIR:=release

GO_TOP := $(shell echo ${GOPATH} | cut -d ':' -f1)

LOCAL_ARCH := $(shell uname -m)
ifeq ($(LOCAL_ARCH),x86_64)
GOARCH_LOCAL := amd64
else
GOARCH_LOCAL := $(LOCAL_ARCH)
endif
export GOARCH ?= $(GOARCH_LOCAL)

export ISTIO_OUT:=$(GO_TOP)/out/$(GOOS)_$(GOARCH)/$(BUILDTYPE_DIR)

docker: docker.pinger docker.svcA docker.svcB docker.svcC

docker.svcA: svcA/Dockerfile
	cd svcA && go build && docker build -t $(HUB)/svc-a:$(TAG) -f Dockerfile .

docker.svcB: svcB/Dockerfile
	cd svcB && go build && docker build -t $(HUB)/svc-b:$(TAG) -f Dockerfile .

docker.svcC: svcC/Dockerfile
	cd svcC && go build && docker build -t $(HUB)/svc-c:$(TAG) -f Dockerfile .

docker.pinger: pinger/Dockerfile
	cd pinger && go build && docker build -t $(HUB)/pinger:$(TAG) -f Dockerfile .

docker.push:
	gcloud docker -- push $(HUB)/svc-a:$(TAG)
	gcloud docker -- push $(HUB)/svc-b:$(TAG)
	gcloud docker -- push $(HUB)/svc-c:$(TAG)
	gcloud docker -- push $(HUB)/pinger:$(TAG)

deploy:
	kubectl apply -f <(${ISTIO_OUT}/istioctl kube-inject -f yaml/svc.yaml --hub=${ISTIO_HUB} --tag=${ISTIO_TAG}) && \
	kubectl apply -f yaml/pinger.yaml && \
	kubectl apply -f yaml/metric.yaml && \
	kubectl apply -f yaml/trace.yaml

clean:
	kubectl delete -f yaml/svc.yaml && \
	kubectl delete -f yaml/pinger.yaml && \
	kubectl delete -f yaml/metric.yaml
