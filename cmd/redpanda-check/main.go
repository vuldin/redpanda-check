package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vuldin/redpanda-check/internal/admin"
	"github.com/vuldin/redpanda-check/internal/checker"
	"github.com/vuldin/redpanda-check/internal/checks"
	"github.com/vuldin/redpanda-check/internal/k8s"
	"k8s.io/client-go/kubernetes"
)

var version = "dev"

func main() {
	var (
		adminURLs     []string
		tlsCA         string
		tlsCert       string
		tlsKey        string
		tlsSkipVerify bool
		saslUser      string
		saslPass      string
		namespace     string
		kubeconfig    string
		format        string
		verbose       bool
		showVersion   bool
	)

	root := &cobra.Command{
		Use:   "redpanda-check",
		Short: "Production readiness checks for Redpanda deployments",
		Long: `Validates a Redpanda deployment against production readiness standards.

Checks cluster health, configuration, resource allocation, and operational
best practices. Connects to the Redpanda admin API directly.

By default only failed and warning checks are shown. Use -v to see all results.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				fmt.Println(version)
				return nil
			}
			return run(cmd.Context(), runConfig{
				adminURLs:     adminURLs,
				tlsCA:         tlsCA,
				tlsCert:       tlsCert,
				tlsKey:        tlsKey,
				tlsSkipVerify: tlsSkipVerify,
				saslUser:      saslUser,
				saslPass:      saslPass,
				namespace:     namespace,
				kubeconfig:    kubeconfig,
				format:        format,
				verbose:       verbose,
			})
		},
	}

	f := root.Flags()
	f.StringSliceVar(&adminURLs, "admin-url", nil, "Admin API addresses (comma-separated)")
	f.StringVar(&tlsCA, "admin-tls-ca", "", "TLS CA certificate path")
	f.StringVar(&tlsCert, "admin-tls-cert", "", "TLS client certificate path")
	f.StringVar(&tlsKey, "admin-tls-key", "", "TLS client key path")
	f.BoolVar(&tlsSkipVerify, "admin-tls-skip-verify", false, "Skip TLS certificate verification (insecure)")
	f.StringVar(&saslUser, "sasl-user", "", "SASL username for admin API basic auth")
	f.StringVar(&saslPass, "sasl-password", "", "SASL password")
	f.StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace (enables K8s checks)")
	f.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	f.StringVar(&format, "format", "text", "Output format: text, json")
	f.BoolVarP(&verbose, "verbose", "v", false, "Show all checks including passing")
	f.BoolVar(&showVersion, "version", false, "Print version and exit")

	// Handle --help-autocomplete for rpk plugin integration.
	if len(os.Args) > 1 && os.Args[1] == "--help-autocomplete" {
		printAutocomplete()
		return
	}

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

type runConfig struct {
	adminURLs     []string
	tlsCA         string
	tlsCert       string
	tlsKey        string
	tlsSkipVerify bool
	saslUser      string
	saslPass      string
	namespace     string
	kubeconfig    string
	format        string
	verbose       bool
}

func run(ctx context.Context, cfg runConfig) error {
	ns := resolveNamespace(cfg.namespace)

	// Attempt K8s client setup for discovery and/or K8s-specific checks.
	var k8sClient kubernetes.Interface
	var discoveredAddrs []string

	if len(cfg.adminURLs) == 0 || cfg.namespace != "" {
		cl, err := k8s.NewClient(cfg.kubeconfig)
		if err == nil {
			k8sClient = cl
			addrs, err := k8s.DiscoverAdminAddresses(ctx, cl, ns)
			if err == nil && len(addrs) > 0 {
				discoveredAddrs = addrs
			}
		}
	}

	if len(cfg.adminURLs) == 0 {
		if len(discoveredAddrs) > 0 {
			cfg.adminURLs = discoveredAddrs
			fmt.Fprintf(os.Stderr, "Discovered Redpanda in namespace %q with %d brokers\n", ns, len(cfg.adminURLs))
		} else {
			return fmt.Errorf("no --admin-url provided and Kubernetes discovery failed; specify --admin-url")
		}
	}

	if cfg.tlsSkipVerify {
		fmt.Fprintf(os.Stderr, "WARNING: TLS certificate verification is disabled. Do not use --admin-tls-skip-verify in production.\n")
	}

	// Build admin API client.
	adminClient, err := admin.NewClient(admin.Config{
		Addresses:     cfg.adminURLs,
		TLSCa:        cfg.tlsCA,
		TLSCert:      cfg.tlsCert,
		TLSKey:       cfg.tlsKey,
		TLSSkipVerify: cfg.tlsSkipVerify,
		SASLUser:      cfg.saslUser,
		SASLPassword:  cfg.saslPass,
	})
	if err != nil {
		return fmt.Errorf("unable to create admin API client: %v", err)
	}
	defer adminClient.Close()

	// Build the checker with shared clients.
	pc := &checker.ProductionChecker{
		AdminClient: adminClient,
		K8sClient:   k8sClient,
		Namespace:   ns,
	}

	// Run all checks, organized by level and type.
	allChecks := []checker.Check{
		// Critical checks (must pass for production readiness).
		checks.ClusterHealth,
		checks.License,
		checks.EnterpriseLicense,
		checks.BrokerCount,
		checks.BrokerMembership,
		checks.MaintenanceMode,
		checks.DeveloperMode,
		checks.Overprovisioned,
		checks.ReplicationFactor,
		checks.MinTopicReplications,
		checks.ExistingTopicsReplication,
		checks.VersionConsistency,
		checks.Authorization,
		checks.TLSEnabled,
		checks.InternalRPCTLS,
		// Recommended checks (best practices).
		checks.DataBalancing,
		checks.RackAwareness,
		checks.TieredStorage,
		checks.AuditLogging,
		checks.CoreBalancing,
		checks.PartitionBalancerStatus,
		checks.BallastFile,
		// K8s-specific checks (skip gracefully when no K8s client).
		checks.PersistentStorage,
		checks.ResourceLimits,
		checks.PodDisruptionBudget,
		checks.CPUMemoryRatio,
		checks.NoFractionalCPU,
		checks.TopologySpread,
		checks.NodeIsolation,
		checks.TuningInitContainer,
	}

	for _, check := range allChecks {
		check(ctx, pc)
	}

	// Generate and print report.
	report := checker.NewReport(pc.Results)

	switch strings.ToLower(cfg.format) {
	case "json":
		if err := checker.PrintJSON(os.Stdout, report); err != nil {
			return fmt.Errorf("unable to print JSON report: %v", err)
		}
	default:
		checker.PrintText(os.Stdout, report, cfg.verbose, k8sClient != nil)
	}

	if report.OverallStatus == checker.StatusFail {
		os.Exit(1)
	}
	return nil
}

func resolveNamespace(ns string) string {
	if ns != "" {
		return ns
	}
	if env := os.Getenv("NAMESPACE"); env != "" {
		return env
	}
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		return strings.TrimSpace(string(data))
	}
	return "redpanda"
}

type pluginHelp struct {
	Path    string   `json:"path"`
	Short   string   `json:"short"`
	Long    string   `json:"long"`
	Example string   `json:"example"`
	Args    []string `json:"args"`
}

func printAutocomplete() {
	helps := []pluginHelp{
		{
			Path:  "check",
			Short: "Production readiness checks for Redpanda deployments",
			Long:  "Validates a Redpanda deployment against production readiness standards.",
		},
	}
	enc := json.NewEncoder(os.Stdout)
	enc.Encode(helps)
}
