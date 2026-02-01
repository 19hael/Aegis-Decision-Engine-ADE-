package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	serverURL string
	rootCmd   = &cobra.Command{
		Use:   "ade-cli",
		Short: "ADE Command Line Interface",
		Long:  `CLI tool for interacting with the Aegis Decision Engine`,
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&serverURL, "server", "s", "http://localhost:8080", "ADE server URL")
	
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(ingestCmd)
	rootCmd.AddCommand(evaluateCmd)
	rootCmd.AddCommand(simulateCmd)
	rootCmd.AddCommand(decisionsCmd)
	rootCmd.AddCommand(actionsCmd)
}

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check server health",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := http.Get(serverURL + "/health")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
		return nil
	},
}

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingest a metrics event",
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceID, _ := cmd.Flags().GetString("service")
		cpu, _ := cmd.Flags().GetFloat64("cpu")
		latency, _ := cmd.Flags().GetFloat64("latency")
		errorRate, _ := cmd.Flags().GetFloat64("error-rate")
		rps, _ := cmd.Flags().GetFloat64("rps")

		event := map[string]interface{}{
			"event_id":        fmt.Sprintf("evt-%d", time.Now().Unix()),
			"idempotency_key": fmt.Sprintf("idemp-%d", time.Now().Unix()),
			"service_id":      serviceID,
			"event_type":      "metrics",
			"payload": map[string]interface{}{
				"cpu":                 cpu,
				"latency_ms":          latency,
				"error_rate":          errorRate,
				"requests_per_second": rps,
				"queue_depth":         0,
			},
			"timestamp": time.Now().Format(time.RFC3339),
		}

		return postJSON("/ingest", event)
	},
}

var evaluateCmd = &cobra.Command{
	Use:   "evaluate",
	Short: "Evaluate a decision",
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceID, _ := cmd.Flags().GetString("service")
		cpu, _ := cmd.Flags().GetFloat64("cpu")
		latency, _ := cmd.Flags().GetFloat64("latency")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		req := map[string]interface{}{
			"service_id": serviceID,
			"features": map[string]interface{}{
				"cpu_current":  cpu,
				"latency_p95":  latency,
				"error_rate":   0.05,
				"health_score": 0.8,
			},
			"dry_run":        dryRun,
			"idempotency_key": fmt.Sprintf("cli-%d", time.Now().Unix()),
		}

		return postJSON("/evaluate", req)
	},
}

var simulateCmd = &cobra.Command{
	Use:   "simulate",
	Short: "Run a simulation",
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceID, _ := cmd.Flags().GetString("service")
		scenario, _ := cmd.Flags().GetString("scenario")
		horizon, _ := cmd.Flags().GetInt("horizon")

		req := map[string]interface{}{
			"service_id":       serviceID,
			"policy_id":        "autoscale_policy",
			"policy_version":   "1.0",
			"scenario":         scenario,
			"horizon_minutes":  horizon,
			"iterations":       100,
			"current_state": map[string]interface{}{
				"cpu_current": 75.0,
				"latency_p95": 450.0,
				"error_rate":  0.05,
			},
		}

		return postJSON("/simulations/run", req)
	},
}

var decisionsCmd = &cobra.Command{
	Use:   "decisions",
	Short: "List decisions",
	RunE: func(cmd *cobra.Command, args []string) error {
		return getJSON("/decisions")
	},
}

var actionsCmd = &cobra.Command{
	Use:   "actions",
	Short: "Execute an action",
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceID, _ := cmd.Flags().GetString("service")
		actionType, _ := cmd.Flags().GetString("type")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		req := map[string]interface{}{
			"action_id":      fmt.Sprintf("act-%d", time.Now().Unix()),
			"action_type":    actionType,
			"target_service": serviceID,
			"payload": map[string]interface{}{
				"urgency": "normal",
			},
			"dry_run": dryRun,
		}

		return postJSON("/actions/execute", req)
	},
}

func postJSON(endpoint string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	resp, err := http.Post(serverURL+endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err == nil {
		fmt.Println(prettyJSON.String())
	} else {
		fmt.Println(string(body))
	}

	return nil
}

func getJSON(endpoint string) error {
	resp, err := http.Get(serverURL + endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err == nil {
		fmt.Println(prettyJSON.String())
	} else {
		fmt.Println(string(body))
	}

	return nil
}

func main() {
	// Setup flags
	ingestCmd.Flags().StringP("service", "S", "api-gateway", "Service ID")
	ingestCmd.Flags().Float64P("cpu", "c", 75.0, "CPU percentage")
	ingestCmd.Flags().Float64P("latency", "l", 450.0, "Latency in ms")
	ingestCmd.Flags().Float64P("error-rate", "e", 0.05, "Error rate")
	ingestCmd.Flags().Float64P("rps", "r", 1000.0, "Requests per second")

	evaluateCmd.Flags().StringP("service", "S", "api-gateway", "Service ID")
	evaluateCmd.Flags().Float64P("cpu", "c", 75.0, "CPU percentage")
	evaluateCmd.Flags().Float64P("latency", "l", 450.0, "Latency in ms")
	evaluateCmd.Flags().BoolP("dry-run", "d", true, "Dry run mode")

	simulateCmd.Flags().StringP("service", "S", "api-gateway", "Service ID")
	simulateCmd.Flags().StringP("scenario", "s", "normal", "Scenario (normal, high_load, failure)")
	simulateCmd.Flags().IntP("horizon", "H", 10, "Horizon in minutes")

	actionsCmd.Flags().StringP("service", "S", "api-gateway", "Service ID")
	actionsCmd.Flags().StringP("type", "t", "scale_up", "Action type")
	actionsCmd.Flags().BoolP("dry-run", "d", true, "Dry run mode")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
