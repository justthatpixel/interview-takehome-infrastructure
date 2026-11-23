# Results

Real numbers from `make check` runs on the local k3d cluster (Docker Desktop,
Apple Silicon). Newest first within each section.

## Current chart (Envoy gateway on, minReplicas 6)

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
| 4 | + Traefik retry middleware | FAIL, 1-2 × 502 per run | access logs: every 502 was in flight on the killed pod, RetryAttempts 0. Traefik 2.11 never resends a written request |
| 4 | + Traefik ServersTransport, keep-alive off (reverted) | FAIL, 3 × 502 | wrong theory, made it worse |
| 4 | + Envoy gateway, minReplicas 3 | FAIL, 4 of 95 (503 × 4) | retries worked (10/10), but 2 surviving pods hit the 20 limit |
| 4 | + minReplicas 6 | PASS | |

## Timings measured

- SIGKILL to EndpointSlice not-ready: about 0.3 s.
- SIGKILL to Traefik dropping the pod from its backends: about 1.8 s
  (Traefik's provider throttle, default 2 s).

## Not yet done

- Repeated runs (5× per environment) to show it is reliable, not lucky.
- Fresh-cluster test (`make down && make cluster && make setup`, then both
  checks).
