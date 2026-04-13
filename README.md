# redpanda-check

Production readiness validation for Redpanda deployments. Connects to the Redpanda admin API and checks cluster health, configuration, resource allocation, and operational best practices.

Designed to run as a standalone binary or as an [rpk managed plugin](https://docs.redpanda.com/current/reference/rpk/).

## Install

```
go install github.com/vuldin/redpanda-check/cmd/redpanda-check@latest
```

Or build from source:

```
git clone https://github.com/vuldin/redpanda-check.git
cd redpanda-check
go build -o redpanda-check ./cmd/redpanda-check
```

## Usage

Point it at your Redpanda admin API:

```
redpanda-check --admin-url localhost:9644
```

For Kubernetes deployments, pass a namespace to enable K8s-specific checks (resource limits, PDBs, storage validation):

```
redpanda-check --admin-url localhost:9644 --namespace redpanda
```

If you omit `--admin-url`, the tool attempts to discover admin API endpoints from Kubernetes using the current kubeconfig context:

```
redpanda-check --namespace redpanda
```

### Authentication

```
redpanda-check --admin-url localhost:9644 \
  --sasl-user admin --sasl-password secret

redpanda-check --admin-url localhost:9644 \
  --admin-tls-ca /path/to/ca.crt \
  --admin-tls-cert /path/to/client.crt \
  --admin-tls-key /path/to/client.key
```

### Output

By default only failures and warnings are printed. Use `-v` to see all results:

```
redpanda-check --admin-url localhost:9644 -v
```

JSON output for CI/CD pipelines:

```
redpanda-check --admin-url localhost:9644 --format json
```

The process exits with code 1 if any critical check fails.

## Checks

All checks run against the admin API and work regardless of how Redpanda was deployed (Helm, operator, Ansible, bare metal). Checks that require Kubernetes are skipped when no `--namespace` is provided.

Based on the Redpanda production readiness checklists:
- [Bare metal / VM](https://docs.redpanda.com/current/deploy/redpanda/manual/production/production-readiness/)
- [Kubernetes](https://docs.redpanda.com/current/deploy/redpanda/kubernetes/k-production-readiness/)

### Critical

Must pass for production readiness. Failures exit with code 1.

| Check | What it validates |
|-------|-------------------|
| cluster_health | All brokers healthy, no leaderless/under-replicated partitions |
| license_check | Valid license loaded and not expired |
| enterprise_license_compliance | No enterprise features used without valid license |
| broker_count | At least 3 brokers |
| broker_membership | All brokers active (not decommissioning) |
| maintenance_mode | No brokers in maintenance mode |
| developer_mode | Developer mode disabled |
| overprovisioned | Overprovisioned mode disabled |
| replication_factor | Default topic replication factor >= 3 |
| min_topic_replications | Minimum topic replications >= 3 |
| existing_topics_replication | All topics have replication factor >= 3 |
| version_consistency | Same Redpanda version on all brokers |
| authorization_enabled | Authorization/SASL enabled |
| tls_enabled | TLS on Kafka API and Admin API listeners (checked per-broker) |
| internal_rpc_tls | TLS on internal RPC (checked per-broker) |
| persistent_storage | PVCs present, no hostPath provisioner (K8s) |
| resource_limits | CPU/memory requests equal limits (K8s) |
| pod_disruption_budget | PDB exists in namespace (K8s) |
| cpu_memory_ratio | Memory-to-CPU ratio >= 2:1 (K8s) |
| no_fractional_cpu | Whole-integer CPU allocations (K8s) |

### Recommended

Best practices. Failures produce warnings but do not affect exit code.

| Check | What it validates |
|-------|-------------------|
| continuous_data_balancing | partition_autobalancing_mode is "continuous" |
| rack_awareness | Rack awareness enabled, all brokers have rack IDs |
| tiered_storage | Cloud storage enabled |
| audit_logging | Audit logging enabled |
| core_balancing | Core balancing on core count change enabled |
| partition_balancer_status | Partition balancer not stalled |
| ballast_file | Ballast file configured (checked per-broker) |
| topology_spread | topologySpreadConstraints or podAntiAffinity configured (K8s) |
| node_isolation | Dedicated node scheduling via nodeSelector/tolerations (K8s) |
| tuning_init_container | Tuning init container completed on all pods (K8s) |

## rpk plugin integration

To use as an rpk managed plugin, place the binary in `~/.local/bin/`:

```
cp redpanda-check ~/.local/bin/.rpk.managed-check
```

Then `rpk check` will discover and execute it, injecting admin API connection details from your rpk profile.

## Testing

```
go test ./...
```
