# Results

Real numbers from `make check` runs on the local k3d cluster (Docker Desktop,
Apple Silicon). Newest first within each section.

## Traefik Buffering, current chart — 2026-11-24

Tested against Traefik **2.11.18** on the existing k3d cluster, using source
commit `726fae5bcdad9e52f9014d4a0b37f84e10869063` plus a copied chart with:

- Envoy disabled and Traefik routing directly to the API Service.
- Buffering `retryExpression: "IsNetworkError() && Attempts() < 3"`.
- Unchanged `minReplicas: 6`, `maxReplicas: 20`, `targetInFlight: 8`.
- Unchanged probes, 15 s preStop, 10 s shutdown timeout, PDB and zone spread.

Each environment completed one full grader run. These are existing-cluster
runs, not fresh-cluster or long-soak results. The final chart removes the
unused Envoy templates and values; its rendered resources were checked
against the tested candidate before adoption.

| Environment | Level 1 | Level 2 | Level 3 | Level 4 | Peak API pods |
|---|---|---|---|---|---|
| development | 0 of 566 | 0 of 611 | 0 of 7006 | 0 of 523 | 12 |
| production | 0 of 596 | 0 of 597 | 0 of 7006 | 0 of 530 | 13 |

[Development log](verification/traefik-buffering/development.log),
[production log](verification/traefik-buffering/production.log).
A separate development level-4 run started at the six-pod minimum and passed
with **0 of 534 failed** ([log](verification/traefik-buffering/baseline-kill.log)).
The peak API counts match the earlier recorded Envoy runs, while removing
three gateway pods per environment. This is a component-count comparison,
not a controlled performance benchmark.

### Interrupted requests: stricter than the grader

The grader excludes requests already running on a killed container. A
separate test sent 50 concurrent GET requests, measured the pods' in-flight
metrics, then sent SIGKILL directly to a busy API container through the node's
container runtime. It counted every client response, including interrupted
requests. The victims had eight or nine active requests immediately before
the kill.

| Replay setting | Kills | Requests | Failed |
|---|---|---|---|
| Disabled (`Attempts() < 1`), negative control | 1 | 50 | 7 (all 502) |
| Enabled (`Attempts() < 3`) | 3 | 150 | 0 |

[Control results](verification/traefik-buffering/strict-control.json),
[replay results](verification/traefik-buffering/strict-replay.json).
The longest response with replay enabled was 3.7294 s. The new passes do not
explain the older Envoy run's 37 unexpected 503s, recorded below.

## Historical Envoy chart: fresh-cluster runs

The original notes disagreed on the minimum replicas for these runs (four
in the heading, six after a rollback). Their exact replica configuration
cannot be confirmed from the old notes. The new Buffering runs above have
an explicit configuration and retained logs.


`make down && make cluster && make setup`, then 3 full `make check` runs per
environment, alternating, 2026-11-23 22:18-23:25. Setup from the repo alone
worked first time.

| Run | development | production |
|---|---|---|
| 1 | PASS all 4 (L3 peak 12 pods, L4 0 of 878) | FAIL level 3: 37 of 5361 (503 × 37) |
| 2 | PASS all 4 (L3 peak 12, L4 0 of 2339) | PASS all 4 (L3 peak 13, L4 0 of 766) |
| 3 | PASS all 4 (L3 peak 12, L4 0 of 507) | FAIL level 2: 11 of 433 (503 × 11) |

Levels 1 and 2 passed in every run except production run 3.

**Production run 3: the laptop slept.** `pmset -g log` shows "Clamshell
Sleep" (lid closed) at 22:48:20, during level 2, until about 23:25. The
grader's clock shows under 2 minutes for a run that took 38 minutes of wall
time. The Docker VM froze with it. Not a chart failure, but not a pass.

**Production run 1: not explained yet.** No sleep in the power log. At
60 req/s the API had already scaled to 13 pods and run 50 s with no
failures; then the grader's own throughput fell (about 190 requests in a 5 s
window instead of about 300) and 32 requests failed at once. The production
gateways' counters (covering all three production runs, the sleep included)
show 34 `per_try_timeout` (a pod took over 5 s, which a healthy pod never
does: requests take at most 3 s), 48 upstream 503s and 31 ejections. That
points to the whole cluster stalling rather than too few pods, possibly
memory pressure in the 7.6 GiB Docker VM (it had 150-470 MiB free at times).
Not proven.

**Production rerun, Mac kept awake** (`caffeinate`, lid open), 3 full runs
back to back on the same cluster, 2026-11-24 05:41-06:01:

| Run | L1 | L2 | L3 | L4 |
|---|---|---|---|---|
| 1 | 0 of 578 | 0 of 2139 | 0 of 7007, peak 13 pods | 0 of 1266 |
| 2 | 0 of 563 | 0 of 665 | 0 of 7008, peak 13 pods | 0 of 875 |
| 3 | 0 of 563 | 0 of 664 | 0 of 7008, peak 13 pods | 0 of 532 |

All passed. The Docker VM's available memory, logged every 5 s, never went
below 1,586 MiB (load average up to 36 during image builds), so these runs
show no memory pressure. The cause of the first production run 1 failure is
still open; it did not repeat in 3 runs.

Totals for these historical Envoy runs: development 3 of 3,
production 4 of 6 (one run lost to a closed lid, one unexplained).

An earlier fresh-cluster attempt with minReplicas 4 (target 8) failed level 3
on both environments (development 503 × 1, production 503 × 4); minReplicas
went back to 6.

## Autoscaling tuning (B2)

Level 3 was peaking at exactly maxReplicas (20) with target 5, so the
ceiling was the limit. Runs on development, minReplicas 6, maxReplicas 20,
one run each:

| Target in flight | Level 3 | Peak pods |
|---|---|---|
| 5 (before) | PASS, 0 of 7008 | 20 (= max) |
| 6 | PASS, 0 of 7005 | 17 |
| 8 | PASS, 0 of 7008 | 12 |
| 10 | FAIL, 1 of 2186 (503 × 1) during the ramp | |

Target 8 passed with Envoy in these runs. The new Buffering results above
also pass without Envoy at a six-pod minimum. These were different
configurations, not a controlled load-balancer comparison.

minReplicas against level 4 alone, target 8, development:

| minReplicas | Level 4 |
|---|---|
| 3 (target 5, before) | FAIL, 4 of 95 (503 × 4) |
| 4 | PASS, 0 of 1061 |
| 5 | PASS, 0 of 586 |
| 6 | PASS, 0 of 908 |

Four replicas passed the standalone level-4 test. Later fresh-cluster
level-3 failures led back to six replicas, as recorded above.

## Cluster load (add-ons)

The server node hit about 250-320% CPU (`docker stats`) and the grader's
kill timed out once. Cause: Prometheus's default scrape jobs pulled about
90,000 samples every 5 s through the k3s API server (`kubernetes-nodes`
48.6k, `kubernetes-api-servers` 41.2k, cAdvisor 8.1k), none of them used.

| | Before | After (pods job only, 2 h retention) |
|---|---|---|
| Server node CPU (`docker stats`, idle) | ~320% | ~16% |
| Prometheus memory | 771Mi | 116Mi |
| Prometheus CPU | 149m | 32m |
| Series in memory | 194,053 | 8,421 |
| Envoy series per gateway pod | ~1,050 | 342 (`usedonly`) |

## Earlier chart (Envoy gateway on, target 5, minReplicas 6)

Full `make check`, one run per environment, 2026-11-23:

| Level | development | production |
|---|---|---|
| 1: deploy under load | PASS, 0 of 597 failed | PASS, 0 of 583 failed |
| 2: two nodes drained | PASS, 0 of 689 failed | PASS, 0 of 669 failed |
| 3: load 14 → 60 req/s | PASS, 0 of 7008 failed, peak 20 pods | PASS, 0 of 7007 failed, peak 20 pods |
| 4: containers SIGKILLed | PASS, 0 of 1105 failed | see below |

Production level 4: the first full run had 0 of 468 failed, but the grader
could not finish its kill. `crictl` StopContainer timed out on the node
(DeadlineExceeded) while the Docker VM was busy (server node at about 250%
CPU). Re-run with `-level 4`: PASS, 0 of 1056 failed.

Level 4 alone on development after raising minReplicas to 6: PASS twice
(0 of 776, 0 of 574).

Envoy counters during a development level 4 run: 10 retries, 10 retry
successes (`envoy_cluster_upstream_rq_retry` / `_retry_success`).

## How each level failed before its fix

| Level | Chart at the time | Result | Cause |
|---|---|---|---|
| 1 | first chart (probes, preStop, maxUnavailable 0) | PASS, 0 of 386 | |
| 2 | no PDB | FAIL, 4 of 41 (503 × 4) | both drained nodes' pods evicted at once, the last pod hit its 20-request limit |
| 2 | + PDB, zone spread | PASS, 0 of 385 | |
| 3 | no autoscaling | FAIL, 2 of 576 (503 × 2) at about 29 req/s | 3 pods cannot hold the load |
| 3 | KEDA, target 8 in flight, max 15 | PASS once (0 of 7006, peak 14 pods), then FAIL 248 of 2000 (503 × 248) | scale-up lags 20-30 s; busiest pod hit 20 while the average was 12 |
| 3 | target 5, max 20 | PASS, 0 of 7007, peak 20 pods | |
| 4 | no retry | FAIL, 1 of 40 (502 × 1) | request in flight on the killed pod |
| 4 | + Traefik retry middleware | FAIL, 1-2 × 502 per run | access logs: every 502 was in flight on the killed pod, RetryAttempts 0. Traefik 2.11's ordinary Retry middleware never resends a written request |
| 4 | + Traefik ServersTransport, keep-alive off (reverted) | FAIL, 3 × 502 | wrong theory, made it worse |
| 4 | + Envoy gateway, minReplicas 3 | FAIL, 4 of 95 (503 × 4) | retries worked (10/10), but 2 surviving pods hit the 20 limit |
| 4 | + minReplicas 6 | PASS | |

## Timings measured

- SIGKILL to EndpointSlice not-ready: about 0.3 s.
- SIGKILL to Traefik dropping the pod from its backends: about 1.8 s
  (Traefik's provider throttle, default 2 s).

