CLUSTER := takehome
CONTEXT := k3d-$(CLUSTER)
REGISTRY := localhost:5001
IMAGE   ?= $(REGISTRY)/proxy-api
TAG     ?= dev
TRAEFIK_PORT ?= 8081
ENV     ?= production
CHART   ?= deploy/api
# Your values files for ENV, comma-separated and in order. Empty uses only the
# chart's values.yaml.
VALUES  ?= $(CHART)/$(ENV).yaml
OS      := $(shell uname -s | tr A-Z a-z)
ARCH    := $(subst aarch64,arm64,$(subst x86_64,amd64,$(shell uname -m)))

.PHONY: cluster setup image check down

# Create the k3d cluster and its registry, with Traefik on localhost:$(TRAEFIK_PORT).
# Then run Postgres in the db namespace with Terraform, seeded from the API's
# init.sql, with one user for each environment.
cluster:
	k3d cluster create --config local/k3d.yaml --port "$(TRAEFIK_PORT):80@loadbalancer" --wait
	terraform -chdir=local/terraform init -input=false
	terraform -chdir=local/terraform apply -input=false -auto-approve -var kube_context=$(CONTEXT)

# Prepare the cluster for the chart (DB password Secrets in the app
# namespaces). Run once after `make cluster`.
setup:
	CONTEXT=$(CONTEXT) local/setup.sh

# Build the API image and push it to the local registry. Set TAG for each new version.
image:
	docker build -t $(IMAGE):$(TAG) services/api
	docker push $(IMAGE):$(TAG)

# Deploy a new image under load, then run the levels in order. Stops at the
# first level that fails.
check:
	bin/check-$(OS)-$(ARCH) -env $(ENV) -chart $(CHART) -values "$(VALUES)" -image $(IMAGE) -cluster $(CLUSTER) -port $(TRAEFIK_PORT)

# Delete the cluster, its registry and everything in them.
down:
	k3d cluster delete $(CLUSTER)
