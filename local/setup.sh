#!/bin/sh
# Prepares the cluster for the chart. Run once after `make cluster`.
#
# Pods can only mount Secrets from their own namespace, so this copies each
# environment's DB password from the db namespace into the namespace the API
# runs in. In a real cluster the External Secrets Operator would create these
# Secrets from Scaleway Secret Manager instead.
set -eu

CONTEXT=${CONTEXT:-k3d-takehome}

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
