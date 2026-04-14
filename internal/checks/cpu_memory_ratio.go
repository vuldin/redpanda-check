package checks

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/vuldin/redpanda-check/internal/checker"
	k8sutil "github.com/vuldin/redpanda-check/internal/k8s"
)

// CPUMemoryRatio validates the CPU to memory ratio is at least 1:2 (2 GiB per core).
func CPUMemoryRatio(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "cpu_memory_ratio",
		Description: "CPU to memory ratio (1:2 minimum)",
		Level:       checker.LevelCritical,
	}

	ratios, err := cpuMemoryRatios(ctx, pc)
	if err != nil {
		r.Status = checker.StatusFail
		r.Details = err.Error()
		pc.AddResult(r)
		return
	}

	var issues, good []string
	for _, br := range ratios {
		if br.ratio < 2.0 {
			issues = append(issues, br.detail)
		} else {
			good = append(good, br.detail)
		}
	}

	if len(issues) > 0 {
		r.Status = checker.StatusFail
		r.Details = fmt.Sprintf("CPU to memory ratio < 1:2 minimum:\n%s", strings.Join(issues, "\n"))
	} else if len(good) > 0 {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("CPU to memory ratio meets minimum 1:2:\n%s", strings.Join(good, "\n"))
	} else {
		r.Status = checker.StatusSkip
		r.Details = "No brokers found"
	}
	pc.AddResult(r)
}

// CPUMemoryRatioRecommended validates the CPU to memory ratio is at least 1:4
// (4 GiB per core), which is the recommended ratio for production workloads.
func CPUMemoryRatioRecommended(ctx context.Context, pc *checker.ProductionChecker) {
	r := checker.CheckResult{
		Name:        "cpu_memory_ratio_recommended",
		Description: "CPU to memory ratio (1:4 recommended)",
		Level:       checker.LevelRecommended,
	}

	ratios, err := cpuMemoryRatios(ctx, pc)
	if err != nil {
		r.Status = checker.StatusWarn
		r.Details = err.Error()
		pc.AddResult(r)
		return
	}

	var below, good []string
	for _, br := range ratios {
		if br.ratio < 4.0 {
			below = append(below, br.detail)
		} else {
			good = append(good, br.detail)
		}
	}

	if len(below) > 0 {
		r.Status = checker.StatusWarn
		r.Details = fmt.Sprintf("CPU to memory ratio < 1:4 recommended for production:\n%s", strings.Join(below, "\n"))
	} else if len(good) > 0 {
		r.Status = checker.StatusPass
		r.Details = fmt.Sprintf("CPU to memory ratio meets recommended 1:4:\n%s", strings.Join(good, "\n"))
	} else {
		r.Status = checker.StatusSkip
		r.Details = "No brokers found"
	}
	pc.AddResult(r)
}

type brokerRatio struct {
	ratio  float64
	detail string
}

// cpuMemoryRatios returns the memory-per-core ratio for each broker.
// On K8s it reads pod resource requests; on non-K8s it uses the Prometheus
// metrics endpoint (allocated + available memory per shard) and num_cores
// from the broker list.
func cpuMemoryRatios(ctx context.Context, pc *checker.ProductionChecker) ([]brokerRatio, error) {
	if pc.IsK8s() {
		return k8sRatios(ctx, pc)
	}
	return metricsRatios(ctx, pc)
}

func k8sRatios(ctx context.Context, pc *checker.ProductionChecker) ([]brokerRatio, error) {
	pods, err := k8sutil.RedpandaPods(ctx, pc.K8sClient, pc.Namespace)
	if err != nil {
		return nil, fmt.Errorf("Unable to list pods: %v", err)
	}

	var ratios []brokerRatio
	for _, pod := range pods {
		for _, c := range pod.Spec.Containers {
			if c.Name != "redpanda" {
				continue
			}
			req := c.Resources.Requests
			if req.Cpu().IsZero() || req.Memory().IsZero() {
				ratios = append(ratios, brokerRatio{
					ratio:  0,
					detail: fmt.Sprintf("%s: missing CPU or memory requests", pod.Name),
				})
				continue
			}

			cpuCores := float64(req.Cpu().MilliValue()) / 1000.0
			memGiB := float64(req.Memory().Value()) / (1024 * 1024 * 1024)
			ratio := memGiB / cpuCores

			ratios = append(ratios, brokerRatio{
				ratio: ratio,
				detail: fmt.Sprintf("%s: ratio %.1f:1 (CPU: %.1f cores, Memory: %.1f GiB)",
					pod.Name, ratio, cpuCores, memGiB),
			})
		}
	}
	return ratios, nil
}

// metricsRatios queries each broker's Prometheus metrics to compute the
// memory-per-core ratio. Total Redpanda memory per broker is the sum of
// redpanda_memory_allocated_memory + redpanda_memory_available_memory across
// all shards. Core count comes from the broker list API.
func metricsRatios(ctx context.Context, pc *checker.ProductionChecker) ([]brokerRatio, error) {
	brokers, err := pc.BrokerList(ctx)
	if err != nil {
		return nil, fmt.Errorf("Unable to list brokers: %v", err)
	}

	// Try per-broker metrics via ForBroker; fall back to connected broker.
	var ratios []brokerRatio
	queried := 0

	for _, b := range brokers {
		bc, err := pc.AdminClient.ForBroker(ctx, b.NodeID)
		if err != nil {
			continue
		}
		raw, err := bc.PublicMetrics(ctx)
		bc.Close()
		if err != nil {
			continue
		}

		totalMem := parseShardMemory(raw)
		if totalMem <= 0 || b.NumCores <= 0 {
			continue
		}

		memGiB := totalMem / (1024 * 1024 * 1024)
		cpuCores := float64(b.NumCores)
		ratio := memGiB / cpuCores
		ratios = append(ratios, brokerRatio{
			ratio: ratio,
			detail: fmt.Sprintf("broker %d: ratio %.1f:1 (cores: %d, memory: %.1f GiB)",
				b.NodeID, ratio, b.NumCores, memGiB),
		})
		queried++
	}

	// Fallback: query the connected broker's metrics.
	if queried == 0 {
		raw, err := pc.AdminClient.PublicMetrics(ctx)
		if err != nil {
			return nil, fmt.Errorf("Unable to get metrics: %v", err)
		}
		totalMem := parseShardMemory(raw)
		if totalMem <= 0 {
			return nil, fmt.Errorf("Unable to parse memory metrics from broker")
		}
		// Use the first broker's core count as representative.
		if len(brokers) > 0 && brokers[0].NumCores > 0 {
			memGiB := totalMem / (1024 * 1024 * 1024)
			cpuCores := float64(brokers[0].NumCores)
			ratio := memGiB / cpuCores
			ratios = append(ratios, brokerRatio{
				ratio: ratio,
				detail: fmt.Sprintf("broker %d: ratio %.1f:1 (cores: %d, memory: %.1f GiB)",
					brokers[0].NodeID, ratio, brokers[0].NumCores, memGiB),
			})
		}
	}

	return ratios, nil
}

// parseShardMemory sums redpanda_memory_allocated_memory and
// redpanda_memory_available_memory across all shards from Prometheus text.
func parseShardMemory(raw []byte) float64 {
	allocated := 0.0
	available := 0.0

	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "redpanda_memory_allocated_memory{") {
			if v := parseMetricValue(line); v > 0 {
				allocated += v
			}
		} else if strings.HasPrefix(line, "redpanda_memory_available_memory{") {
			if v := parseMetricValue(line); v > 0 {
				available += v
			}
		}
	}
	return allocated + available
}

func parseMetricValue(line string) float64 {
	idx := strings.LastIndex(line, "} ")
	if idx < 0 {
		return 0
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(line[idx+2:]), 64)
	if err != nil {
		return 0
	}
	return v
}
