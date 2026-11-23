#!/bin/sh
# Prepares the cluster for the chart. Run once after `make cluster`.
set -eu

CONTEXT=${CONTEXT:-k3d-takehome}
ADDONS=$(dirname "$0")/addons

# Pods can only mount Secrets from their own namespace, so this copies each
# environment's DB password from the db namespace into the namespace the API
# runs in. In a real cluster the External Secrets Operator would create these
# Secrets from Scaleway Secret Manager instead.
for env in development production; do
  kubectl --context "$CONTEXT" create namespace "$env" --dry-run=client -o yaml \
    | kubectl --context "$CONTEXT" apply -f -

  kubectl --context "$CONTEXT" -n db get secret "$env-postgres-password" \
      -o jsonpath='{.data.password}' \
    | base64 -d \
    | kubectl --context "$CONTEXT" -n "$env" create secret generic postgres-password \
        --from-file=password=/dev/stdin --dry-run=client -o yaml \
    | kubectl --context "$CONTEXT" apply -f -
done

# The API's work is waiting, not CPU, so it scales on requests in flight.
# Prometheus scrapes that metric and KEDA turns it into an HPA.
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts >/dev/null
helm repo add kedacore https://kedacore.github.io/charts >/dev/null
helm repo update prometheus-community kedacore >/dev/null

helm --kube-context "$CONTEXT" upgrade --install prometheus prometheus-community/prometheus \
  --version 29.49.0 --namespace monitoring --create-namespace \
  --values "$ADDONS/prometheus.yaml" --wait

helm --kube-context "$CONTEXT" upgrade --install keda kedacore/keda \
  --version 2.21.1 --namespace keda --create-namespace \
  --values "$ADDONS/keda.yaml" --wait
