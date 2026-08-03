# Envoy routing scenarios

Hands-on kind-based setups for exploring Envoy routing behavior with a static
bootstrap config (no kgateway, no xDS, no control plane). Use these to build
intuition about how zone-aware and priority-based load balancing actually
behave in Envoy.

## Scenarios

1. `01-base` -- single-node cluster, one Envoy proxy, one echo backend. The
   simplest possible setup; useful as a sanity check that your kind + Envoy +
   port-forward loop works before adding routing complexity.
2. `02-zone-aware` -- 3-worker cluster with `topology.kubernetes.io/zone`
   labels, one backend Deployment pinned to each zone, and Envoy configured
   with `common_lb_config.zone_aware_lb_config` including `force_local_zone`.
   Envoy is pinned to `us-east-1a` to match its bootstrap `node.locality`.
3. `03-priority` -- single-node cluster with two backends (primary + fallback)
   exposed as separate `LocalityLbEndpoints` at `priority: 0` and `priority: 1`.
   No zone-aware config; demonstrates priority-based failover.

## Running a scenario

Pick a scenario, create its cluster, apply manifests, port-forward, and probe.

```bash
cd 02-zone-aware
kind create cluster --config kind.yaml
kubectl apply -f backends.yaml
kubectl apply -f envoy-config.yaml -f envoy.yaml
kubectl wait --for=condition=Available deploy --all --timeout=120s
kubectl port-forward svc/envoy 10000:10000 9901:9901 &
```

Generate traffic and tally by backend group (each backend echoes a distinct
string -- `zone-a`, `zone-b`, `zone-c`, `primary`, `fallback`, or `backend`):

```bash
for i in {1..60}; do curl -s localhost:10000/; done | sort | uniq -c
```

Inspect Envoy's view of the world via the admin interface:

```bash
# Zone-aware counters (only emitted when zone_aware_lb_config is set)
curl -s 'localhost:9901/stats?filter=cluster.backend.*zone'

# Per-priority active requests
curl -s 'localhost:9901/stats?filter=cluster.backend.*upstream_rq_active'

# Resolved endpoints with their locality and priority
curl -s 'localhost:9901/config_dump?include_eds' | jq '.configs[] | select(.["@type"] | endswith("EndpointsConfigDump"))'
```

## Things worth toggling per scenario

- `02-zone-aware`: edit `node.locality.zone` in `envoy-config.yaml` to a
  value that doesn't match any backend group (e.g. `us-east-1z`) -- Envoy
  should silently degrade to weighted distribution across all zones even
  with `force_local_zone` set. This reproduces the "mismatched zone string"
  failure mode that motivated the priority-rewrite approach in PR 13978.
- `02-zone-aware`: comment out `force_local_zone` to see prefer-local
  behavior (mostly local, some cross-zone bleed) versus strict-local.
- `02-zone-aware`: scale `backend-a` to zero -- with `force_local_zone`
  Envoy should fall back to the other zones (force is gated on
  `min_endpoints_in_zone_threshold`).
- `03-priority`: scale `backend-primary` to zero -- traffic flips to
  `backend-fallback`. Exercises Envoy's `overprovisioning_factor`
  (default 1.4): P=1 only takes over once P=0 health drops below 1/1.4.

## Cleanup

```bash
kind delete cluster --name envoy-base
kind delete cluster --name envoy-zone-aware
kind delete cluster --name envoy-priority
```

## Image versions

- Envoy: `envoyproxy/envoy:v1.33-latest`
- Echo backend: `hashicorp/http-echo:1.0.0` (listens on port 5678, echoes
  whatever is passed via `-text`)

Both pinned in the manifests; bump as needed.
