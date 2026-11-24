# Proxy API - Infrastructure Take-Home Test

> [!WARNING]
> This repo holds three take-home tests, each on its own branch: `backend`,
> `full-stack` and `infrastructure`. Before you start, ask your contact at
> Byteful which test to do.

The API is a core part of **Byteful**. Our customers use it all day, every
day, to get their list of proxies and to create proxy users. They need every
request to succeed, whatever we deploy and whatever happens to our
infrastructure. So we must take care with how we run it.

## Your goal

Write a **Helm chart** for the API in `services/api`. The API must stay
available while you deploy it, while Kubernetes moves its pods, and while the
load goes up and down.

1. Write the chart in `deploy/api`.
2. Run `make check` until you pass level 1. Then change your chart to pass
   each next level.
3. Write a short README. Explain your work, what you would change for a real
   production cluster, and anything else you want to tell us.

Spend no more than **3 hours**. We prefer a well thought out answer to a
rushed one written by AI.

Submit a GitHub repository, public or private, with your chart and your
README. A short video that explains your solution is a bonus. Good luck!

---

## Set up

We want this take-home to be fun, so we give you a local k3d cluster to work
in. We tested it on Linux. If it does not run on your machine, contact us and
we will give you a development environment.

The local cluster is as close to our production cluster as we could make it.
It uses Traefik as the ingress controller, like ours. In production we deploy
with Argo CD, but Argo CD is hard to run locally, so `make check` runs the
Helm upgrades for you.

You can install other tools that your solution needs in the cluster, for
example Prometheus. You can run them on the control-plane node.

### Prerequisites

| Tool | Install guide |
|---|---|
| Docker | [Get Docker](https://docs.docker.com/get-started/get-docker/) |
| k3d | [Install k3d](https://k3d.io/stable/#installation) |
| kubectl | [Install kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl) |
| Helm | [Install Helm](https://helm.sh/docs/intro/install/) |
| Terraform | [Install Terraform](https://developer.hashicorp.com/terraform/install) |
| make | Your package manager, for example `apt install make` or `xcode-select --install` |

### Your local cluster

```bash
make cluster      # k3d cluster with Traefik on localhost:8081, a registry and Postgres
make check        # deploy your chart under load, then run the levels
make down         # delete the cluster
```

If port `8081` is in use, run `make cluster TRAEFIK_PORT=18081`.

The cluster has a control-plane node, which your API pods cannot run on, and
three workers in different zones:

| Node | Zone (`topology.kubernetes.io/zone`) |
|---|---|
| `k3d-takehome-agent-0` | `ams-1` |
| `k3d-takehome-agent-1` | `par-1` |
| `k3d-takehome-agent-2` | `lon-1` |

Postgres is already running in the cluster:

| Setting | `development` | `production` |
|---|---|---|
| Host | `postgres.db.svc.cluster.local` | `postgres.db.svc.cluster.local` |
| Port | `5432` | `5432` |
| User | `api_development` | `api_production` |
| Database | `ping_residential` | `ping_residential` |
| Password | Secret `db/development-postgres-password` | Secret `db/production-postgres-password` |

In our clusters, the External Secrets Operator creates these Secrets from
Scaleway Secret Manager. Locally they are in the `db` namespace, with the
password in the `password` key. You can change their names, keys and
namespaces.

### Check your work

`make check` stops at the first level that fails. Level 1 deploys a new
image of your chart while requests run. At least one pod must be replaced,
and no request may fail. When you pass a level, the check tells you what the
next level does.

```
level 1: deploy under load
  Deploys a new image of your chart while requests run. At least one pod must be replaced, and no request may fail.
  FAIL  16 of 133 failed (502 × 16)
```

Use `ENV=production` to check the other environment.

You do not need to install the chart yourself. `make check` builds a new
image and runs:

```bash
helm upgrade --install proxy-api deploy/api \
  --namespace <environment> --create-namespace \
  --values <your values files> \
  --set image.repository=localhost:5001/proxy-api \
  --set image.tag=<new tag> \
  --wait --timeout 5m
```

Name and arrange your values files as you like. Tell `make check` which files
to use with `VALUES`, comma-separated and in order. For example:

```bash
make check VALUES=deploy/api/production.yaml ENV=production
```

You can also set `VALUES` in the `Makefile`. If `VALUES` is empty, Helm uses
only the chart's `values.yaml`.

Your chart must follow these rules:

| Setting | Value |
|---|---|
| Chart path | `deploy/api` |
| Release name | `proxy-api` |
| Namespace | `development` or `production` |
| Route | `GET /proxy` on the host `development.localhost` or `production.localhost` |
| Image | Read from `image.repository` and `image.tag` |
| Pod label | `app.kubernetes.io/name: proxy-api` on the API pods |

`make check` installs only your chart. Set up everything else before you run
it.

---

## The API

The API is in `services/api`. Its ports, endpoints, metrics and config are in
[`services/api/README.md`](services/api/README.md).

---

## Use of AI

We are an AI first company. We love AI, but we hate slop. You can use AI
tools. You must still be able to explain every line as if you wrote it
yourself.
