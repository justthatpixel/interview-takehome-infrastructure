# Proxy API: Helm chart

A Helm chart in [`deploy/api`](deploy/api) that keeps the proxy API answering
every request while it is redeployed, while its nodes are drained, while load
triples, and while its containers are killed. All four `make check` levels
pass on both environments with zero failed requests.

The task brief is kept in [`docs/task.md`](docs/task.md). Every number below
comes from real runs, recorded in [`docs/results.md`](docs/results.md).

```
client ─▶ Traefik ─▶ Envoy gateway (3 pods) ─▶ API pods (6-20, autoscaled)
          (Ingress)    retries GET /proxy        │
                       on another pod            ├─▶ Postgres
                                                 └─▶ Prometheus ◀─ KEDA (HPA)
```

## How to run it

Prerequisites are in the task brief (Docker, k3d, kubectl, Helm, Terraform,
make). On macOS the grader binary may be quarantined:
`/usr/bin/xattr -d com.apple.quarantine bin/check-darwin-arm64`.

```bash
make cluster                    # k3d cluster, registry, Postgres (unchanged)
make setup                      # DB Secrets + Prometheus + KEDA (local/setup.sh)
make check ENV=development
make check ENV=production
```

`make setup` is idempotent. It copies each environment's DB password Secret
from the `db` namespace into `development` and `production`, then installs
pinned versions of Prometheus and KEDA on the control-plane node, so a drain
of the workers never stops autoscaling. `VALUES` in the `Makefile` defaults to
`deploy/api/$(ENV).yaml`.

Keep the machine awake during `make check`. One of my runs failed because the
laptop lid closed and froze the Docker VM.

## The chart

| File | What it is |
|---|---|
| `values.yaml` | Shared defaults, with a comment on why each value is what it is |
| `development.yaml`, `production.yaml` | Host and DB user per environment |
| `templates/deployment.yaml` | The API: probes, preStop, rolling update, zone spread |
| `templates/configmap.yaml` | `/root/configs/config.yaml` (the API exits without it) |
| `templates/service.yaml`, `ingress.yaml` | Service, and `GET /proxy` on the environment host |
| `templates/pdb.yaml` | PodDisruptionBudget for the API |
| `templates/scaledobject.yaml` | KEDA autoscaling on requests in flight |
| `templates/retry.yaml` | Traefik retry Middleware |
| `templates/gateway-*.yaml` | Envoy gateway: config, Deployment, Services, PDB |

The YAML comments say why each setting is there. This README says which
failure it fixed.

## Level by level

### Level 1: deploy under load

**Failure without it.** The API closes port 8080 the moment it gets SIGTERM.
Traefik only learns the pod is going away a moment later, and sends requests
to the closed port in between (the brief's example: `502 × 16`).

**Fix.**
- `preStop` sleep (15 s) before SIGTERM, so whatever routes to the pod has
  dropped it before the port closes. 10 s was enough for Traefik alone; the
  gateway finds pods by DNS, which can lag about 6 s, hence 15.
- `terminationGracePeriodSeconds` = preStop + `shutdownTimeout` (10 s) + 10 s
  buffer, computed in the template, so the kubelet never SIGKILLs a drain.
- Probes on the health port 9090: startup `/db` (Postgres answered and 8080
  is open, allowing 60 s), readiness `/status` (503 after SIGTERM), liveness
  `/ping`. Liveness deliberately does not check Postgres, or a DB outage would
  restart every pod at once.
- `maxUnavailable: 0`, `maxSurge: 1`: an old pod goes only after its
  replacement is Ready.
- A config checksum annotation rolls pods when the config changes.

**Result.** 0 of 386 failed on the first run.

### Level 2: two nodes drained at once

**Failure.** `503 × 4`. Both drained nodes' pods were evicted together, and the
single pod left hit the API's 20-request limit.

**Fix.**
- PodDisruptionBudget, `maxUnavailable: 1`. The drain evicts one pod, waits
  for its replacement to be Ready, then the next. `maxUnavailable` rather than
  `minAvailable`, so it stays right as the autoscaler changes the count.
- `topologySpreadConstraints` on zone, `maxSkew: 1`, `DoNotSchedule`, with
  `nodeTaintsPolicy: Honor`. Without `Honor`, cordoned nodes still count
  toward the skew, and the replacements for a drain can sit in Pending.

**Result.** 0 of 385 failed.

### Level 3: load from 14 to 60 req/s

**Failure.** `503 × 2` at about 29 req/s: three pods cannot hold it. CPU stays
near zero because a request is mostly a sleep, so CPU-based autoscaling is
useless here.

**Fix.** Scale on `http_requests_in_flight`, which the API already exports.
- Prometheus scrapes the pods every 5 s (pod annotations). KEDA's
  `ScaledObject` turns the query
  `sum(http_requests_in_flight{namespace, app})` into an HPA that targets an
  average of **8 in flight per pod**, between **6 and 20 pods**.
- HPA `behavior`: scale up by up to 100% every 15 s with no stabilisation;
  scale down by 2 pods per 30 s after a 60 s window. Down is deliberately slow,
  so a short dip does not remove capacity, and every removal is a pod
  shutdown that must drain cleanly.
- The Deployment has no `replicas` field, so a `helm upgrade` never resets
  the count the autoscaler chose.

**How the target was chosen.** A scale-up takes 20-30 s to land (scrape, HPA
sync, pod start). Behind Traefik's round-robin, target 8 failed badly on a
rerun (`503 × 248`): the busiest pod hit 20 while the average was 12, so I
dropped to 5. With the Envoy gateway (below), which sends each request to the
pod with the fewest in flight, I measured again:

| Target | Level 3 | Peak pods |
|---|---|---|
| 5 | pass | 20 (the ceiling) |
| 6 | pass | 17 |
| 8 | pass | 12 |
| 10 | fail, `503 × 1` | |

8 is the highest that passes, and it leaves headroom under `maxReplicas`.

**Result.** 0 of 7,007 failed, peak 12-13 pods.

### Level 4: containers killed with SIGKILL

**Failure.** `502 × 1-2` per run. A killed container gets no preStop and no
drain, so every request it was working on fails.

**What did not work, and why.** I added a Traefik retry Middleware first. It
did not help. Traefik's access logs showed every 502 had `RetryAttempts: 0`
and was a request already in flight on the killed pod: all of them ended at
the same instant, the kill. Traefik 2.11 stops retrying once it has written a
request to the backend, so it cannot save these. I also tried turning off
Traefik's keep-alive to the pods, on the theory that requests were being
written onto dead connections. It made things worse (`502 × 3`) and I
reverted it.

**Fix: an Envoy gateway** (`gateway.enabled`, on by default) between Traefik
and the API:
- Envoy finds the API pods through a headless Service, so it can tell them
  apart. A normal Service would hide them behind one IP.
- `GET /proxy` is retried on `reset`, `connect-failure` and `refused-stream`,
  up to 2 times, on a **different** pod (`previous_hosts`). It is safe because
  the request only reads. It never retries a 5xx: the API answers 503 when
  full, and retrying that would hide overload and add to it.
- A retry budget caps retries at 20% of requests in flight (at least 25, so
  one full pod of 20 can always be retried). Envoy's default allows only 3
  retries at once for the whole cluster.
- Outlier detection ejects a pod after one connection failure (it is dead)
  but never for 503s (it is busy). Ejecting busy pods under load would push
  their load onto the rest.
- `LEAST_REQUEST` load balancing, which is also why target 8 now works.
- 3 pods, zone-spread, with their own PDB and a 20 s preStop, since Envoy
  exits straight away on SIGTERM.
- The admin API is bound to `127.0.0.1`. A separate port (9902) answers only
  `/healthz` (probes) and `/stats/prometheus`, and returns 404 for anything
  else, including `/quitquitquit`.
- The Traefik retry Middleware stays (3 attempts, 50 ms): it covers a
  *gateway* pod dying, where Traefik's connection is refused.

**Second failure, then `minReplicas`.** With the gateway, level 4 failed
`503 × 4` instead. Retries worked (10 of 10 succeeded), but with 3 pods a kill
left 2 to carry about 32 requests plus the retries, and one hit 20. With 6
pods (two per zone), losing a pod or a whole zone still leaves 4. I tried 4:
it passed level 4 but failed level 3 on a fresh cluster, because the ramp
starts from `minReplicas`. So 6.

**Result.** 0 failed in every level 4 run since (507 to 2,339 requests each).

## Results

From a fresh cluster (`make down && make cluster && make setup`), final
chart:

| | Full `make check` runs | Passed |
|---|---|---|
| development | 3 | 3 |
| production, first 3 | 3 | 1 |
| production, rerun with the Mac kept awake | 3 | 3 |

Of the two production failures, one was the laptop lid closing mid-run (the
power log shows the sleep; the grader's clock showed 2 minutes for a
38-minute run). The other (`503 × 37` during the 60 req/s hold) is **not
explained**: the API had already scaled to 13 pods, then the grader's own
throughput dipped and requests failed together, which looks like the whole
cluster stalling rather than too few pods. It did not repeat in the next
three runs. Details in [`docs/results.md`](docs/results.md).

## Trade-offs and known limits

- **The gateway is an extra component and an extra hop.** It is the most
  complex part of the chart, and a retried request takes up to about twice as
  long. I added it only after showing Traefik could not do the job.
- **DB connections.** pgxpool's default maximum is `max(4, CPUs)` per pod: 12
  on these nodes. Both environments at `maxReplicas` would be 40 pods and up
  to 480 connections, against Postgres's 100. In practice a pod holds one or
  two, because it uses a connection only for one short query. The pool size
  cannot be set without changing the app.
- **Pods do not rebalance after a drain.** After level 2 all pods can end up
  on one node; spread only applies when pods are scheduled.
- **Autoscaling depends on Prometheus.** If it is down, the HPA keeps the
  current count but cannot scale up. There is no KEDA `fallback` yet.
- **Idle cost.** `minReplicas: 6` keeps six API pods and three gateway pods
  running at any load. That is the price of surviving a kill at baseline
  traffic.
- **Runs as root.** The image's working directory is `/root`, and I did not
  add a `securityContext`.
- **The add-ons are trimmed for a laptop.** Prometheus scrapes only annotated
  pods with 2 h retention. Its default jobs pulled about 90,000 samples every
  5 s through the k3s API server, and pushed the control-plane node to about
  320% CPU (the grader's container kill timed out once). After trimming:
  about 16% CPU, 116 MiB instead of 771 MiB.

## What I would change for production

- **Retries in the ingress instead of a separate gateway.** An ingress or mesh
  that can retry idempotent requests after they were sent (Envoy Gateway,
  Istio, or a newer ingress controller) would remove the extra hop.
- **Argo CD** to deploy the chart, as Byteful already does.
- **External Secrets Operator** with Scaleway Secret Manager for the DB
  password, instead of `local/setup.sh` copying Secrets.
- **PgBouncer, or a configurable pool size** in the API, to bound DB
  connections properly.
- **Sizing from load tests.** Requests and limits here are small guesses
  (50m / 64Mi, 128Mi limit), and `minReplicas` and the in-flight target should
  come from real traffic.
- **Alerts** on the 5xx rate, Envoy retry rate, container restarts, and
  in-flight per pod close to 20.
- **NetworkPolicies:** API 8080 reachable only from the gateway, the gateway
  only from Traefik, egress only to Postgres and DNS.
- **Hardening:** non-root (needs a Dockerfile change away from `/root`), a
  read-only root filesystem, dropped capabilities, and images pinned by
  digest.
- **More than one Traefik replica.** Here it is a single pod on the control
  plane, as provided.
- **A managed, highly available database** instead of one Postgres pod.
- **The descheduler**, to rebalance pods across zones after a drain.
- **A KEDA `fallback`**, so a Prometheus outage holds a safe replica count.

## Time spent

The chart for levels 1-3 took about 25 minutes. Level 4 took most of the
time: about an hour to find out why Traefik's retries did nothing and to build
the gateway. Tuning, trimming the add-ons and verification took about an hour
more. The fresh-cluster and repeated runs then ran over several hours,
mostly unattended. That is beyond the 3-hour budget; the extra went on making
level 4 correct and proving the result is repeatable.
