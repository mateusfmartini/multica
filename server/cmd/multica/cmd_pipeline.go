package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/multica-ai/multica/server/internal/cli"
)

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Manage pipelines and pipeline columns",
}

var pipelineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pipelines in the workspace",
	RunE:  runPipelineList,
}

var pipelineGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get pipeline details",
	Args:  exactArgs(1),
	RunE:  runPipelineGet,
}

var pipelineCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new pipeline",
	RunE:  runPipelineCreate,
}

var pipelineUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a pipeline",
	Args:  exactArgs(1),
	RunE:  runPipelineUpdate,
}

var pipelineDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a pipeline",
	Args:  exactArgs(1),
	RunE:  runPipelineDelete,
}

var pipelineSetDefaultCmd = &cobra.Command{
	Use:   "set-default <id>",
	Short: "Set pipeline as workspace default",
	Args:  exactArgs(1),
	RunE:  runPipelineSetDefault,
}

var pipelineColumnCmd = &cobra.Command{
	Use:   "column",
	Short: "Manage pipeline columns",
}

var pipelineColumnListCmd = &cobra.Command{
	Use:   "list <pipeline-id>",
	Short: "List columns in a pipeline",
	Args:  exactArgs(1),
	RunE:  runPipelineColumnList,
}

var pipelineColumnAddCmd = &cobra.Command{
	Use:   "add <pipeline-id>",
	Short: "Add a column to a pipeline",
	Args:  exactArgs(1),
	RunE:  runPipelineColumnAdd,
}

var pipelineColumnUpdateCmd = &cobra.Command{
	Use:   "update <pipeline-id> <status-key>",
	Short: "Update a pipeline column",
	Args:  exactArgs(2),
	RunE:  runPipelineColumnUpdate,
}

var pipelineColumnDeleteCmd = &cobra.Command{
	Use:   "delete <pipeline-id> <status-key>",
	Short: "Delete a column from a pipeline",
	Args:  exactArgs(2),
	RunE:  runPipelineColumnDelete,
}

var pipelineColumnSyncCmd = &cobra.Command{
	Use:   "sync <pipeline-id>",
	Short: "Replace all pipeline columns with JSON input from stdin or --file",
	Args:  exactArgs(1),
	RunE:  runPipelineColumnSync,
}

func init() {
	pipelineCmd.AddCommand(pipelineListCmd)
	pipelineCmd.AddCommand(pipelineGetCmd)
	pipelineCmd.AddCommand(pipelineCreateCmd)
	pipelineCmd.AddCommand(pipelineUpdateCmd)
	pipelineCmd.AddCommand(pipelineDeleteCmd)
	pipelineCmd.AddCommand(pipelineSetDefaultCmd)
	pipelineCmd.AddCommand(pipelineColumnCmd)

	pipelineColumnCmd.AddCommand(pipelineColumnListCmd)
	pipelineColumnCmd.AddCommand(pipelineColumnAddCmd)
	pipelineColumnCmd.AddCommand(pipelineColumnUpdateCmd)
	pipelineColumnCmd.AddCommand(pipelineColumnDeleteCmd)
	pipelineColumnCmd.AddCommand(pipelineColumnSyncCmd)

	// pipeline list
	pipelineListCmd.Flags().String("output", "table", "Output format: table or json")

	// pipeline get
	pipelineGetCmd.Flags().String("output", "json", "Output format: table or json")

	// pipeline create
	pipelineCreateCmd.Flags().String("name", "", "Pipeline name (required)")
	pipelineCreateCmd.Flags().String("description", "", "Pipeline description")
	pipelineCreateCmd.Flags().Bool("default", false, "Set as workspace default pipeline")
	pipelineCreateCmd.Flags().String("output", "json", "Output format: table or json")

	// pipeline update
	pipelineUpdateCmd.Flags().String("name", "", "New pipeline name")
	pipelineUpdateCmd.Flags().String("description", "", "New description")
	pipelineUpdateCmd.Flags().String("output", "json", "Output format: table or json")

	// pipeline column list
	pipelineColumnListCmd.Flags().String("output", "table", "Output format: table or json")

	// pipeline column add
	pipelineColumnAddCmd.Flags().String("status-key", "", "Unique status key for this column (no spaces, required)")
	pipelineColumnAddCmd.Flags().String("label", "", "Display label (defaults to status-key)")
	pipelineColumnAddCmd.Flags().Int("position", -1, "Display position (appended last if not set)")
	pipelineColumnAddCmd.Flags().Bool("terminal", false, "Mark column as terminal (workflow ends here)")
	pipelineColumnAddCmd.Flags().String("instructions", "", "Markdown instructions for agents in this column")
	pipelineColumnAddCmd.Flags().StringSlice("allow", nil, "Allowed status transitions (comma-separated status keys)")
	pipelineColumnAddCmd.Flags().String("output", "json", "Output format: table or json")

	// pipeline column update
	pipelineColumnUpdateCmd.Flags().String("label", "", "New display label")
	pipelineColumnUpdateCmd.Flags().Int("position", -1, "New position")
	pipelineColumnUpdateCmd.Flags().Bool("terminal", false, "Set terminal flag")
	pipelineColumnUpdateCmd.Flags().Bool("no-terminal", false, "Clear terminal flag")
	pipelineColumnUpdateCmd.Flags().String("instructions", "", "New instructions (use empty string to clear)")
	pipelineColumnUpdateCmd.Flags().StringSlice("allow", nil, "New allowed transitions (replaces existing)")
	pipelineColumnUpdateCmd.Flags().String("output", "json", "Output format: table or json")

	// pipeline column delete
	// (no flags)

	// pipeline column sync
	pipelineColumnSyncCmd.Flags().String("file", "", "Path to JSON file with columns array (reads stdin if not set)")
	pipelineColumnSyncCmd.Flags().String("output", "json", "Output format: table or json")
}

// ---------------------------------------------------------------------------
// Pipeline commands
// ---------------------------------------------------------------------------

func runPipelineList(cmd *cobra.Command, _ []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if _, err := requireWorkspaceID(cmd); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	path := fmt.Sprintf("/api/workspaces/%s/pipelines", client.WorkspaceID)
	var pipelines []map[string]any
	if err := client.GetJSON(ctx, path, &pipelines); err != nil {
		return fmt.Errorf("list pipelines: %w", err)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, pipelines)
	}

	headers := []string{"ID", "NAME", "DESCRIPTION", "DEFAULT"}
	rows := make([][]string, 0, len(pipelines))
	for _, p := range pipelines {
		isDefault := ""
		if b, _ := p["is_default"].(bool); b {
			isDefault = "✓"
		}
		rows = append(rows, []string{
			truncateID(strVal(p, "id")),
			strVal(p, "name"),
			strVal(p, "description"),
			isDefault,
		})
	}
	cli.PrintTable(os.Stdout, headers, rows)
	return nil
}

func runPipelineGet(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if _, err := requireWorkspaceID(cmd); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	path := fmt.Sprintf("/api/workspaces/%s/pipelines/%s", client.WorkspaceID, args[0])
	var pipeline map[string]any
	if err := client.GetJSON(ctx, path, &pipeline); err != nil {
		return fmt.Errorf("get pipeline: %w", err)
	}

	return cli.PrintJSON(os.Stdout, pipeline)
}

func runPipelineCreate(cmd *cobra.Command, _ []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if _, err := requireWorkspaceID(cmd); err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("--name is required")
	}

	body := map[string]any{"name": name}
	if desc, _ := cmd.Flags().GetString("description"); desc != "" {
		body["description"] = desc
	}
	if isDefault, _ := cmd.Flags().GetBool("default"); isDefault {
		body["is_default"] = true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	path := fmt.Sprintf("/api/workspaces/%s/pipelines", client.WorkspaceID)
	var pipeline map[string]any
	if err := client.PostJSON(ctx, path, body, &pipeline); err != nil {
		return fmt.Errorf("create pipeline: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Pipeline %s created.\n", strVal(pipeline, "id"))
	return cli.PrintJSON(os.Stdout, pipeline)
}

func runPipelineUpdate(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if _, err := requireWorkspaceID(cmd); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Fetch current pipeline to fill defaults.
	path := fmt.Sprintf("/api/workspaces/%s/pipelines/%s", client.WorkspaceID, args[0])
	var current map[string]any
	if err := client.GetJSON(ctx, path, &current); err != nil {
		return fmt.Errorf("get pipeline: %w", err)
	}

	body := map[string]any{
		"name":        strVal(current, "name"),
		"description": strVal(current, "description"),
	}
	if v, _ := cmd.Flags().GetString("name"); v != "" {
		body["name"] = strings.TrimSpace(v)
	}
	if cmd.Flags().Changed("description") {
		body["description"], _ = cmd.Flags().GetString("description")
	}

	var pipeline map[string]any
	if err := client.PatchJSON(ctx, path, body, &pipeline); err != nil {
		return fmt.Errorf("update pipeline: %w", err)
	}

	return cli.PrintJSON(os.Stdout, pipeline)
}

func runPipelineDelete(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if _, err := requireWorkspaceID(cmd); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	path := fmt.Sprintf("/api/workspaces/%s/pipelines/%s", client.WorkspaceID, args[0])
	if err := client.DeleteJSON(ctx, path); err != nil {
		return fmt.Errorf("delete pipeline: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Pipeline %s deleted.\n", args[0])
	return nil
}

func runPipelineSetDefault(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if _, err := requireWorkspaceID(cmd); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	path := fmt.Sprintf("/api/workspaces/%s/pipelines/%s/set-default", client.WorkspaceID, args[0])
	if err := client.PostJSON(ctx, path, nil, nil); err != nil {
		return fmt.Errorf("set default pipeline: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Pipeline %s set as default.\n", args[0])
	return nil
}

// ---------------------------------------------------------------------------
// Pipeline column commands
// ---------------------------------------------------------------------------

func runPipelineColumnList(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if _, err := requireWorkspaceID(cmd); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cols, err := fetchPipelineColumns(ctx, client, args[0])
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, cols)
	}

	headers := []string{"POS", "STATUS KEY", "LABEL", "TERMINAL", "TRANSITIONS", "INSTRUCTIONS"}
	rows := make([][]string, 0, len(cols))
	for _, c := range cols {
		pos := fmt.Sprintf("%v", c["position"])
		terminal := ""
		if b, _ := c["is_terminal"].(bool); b {
			terminal = "✓"
		}
		transitions := ""
		if ts, ok := c["allowed_transitions"].([]any); ok {
			parts := make([]string, 0, len(ts))
			for _, t := range ts {
				parts = append(parts, fmt.Sprintf("%v", t))
			}
			transitions = strings.Join(parts, ", ")
		}
		instr := strVal(c, "instructions")
		if len(instr) > 40 {
			instr = instr[:37] + "..."
		}
		rows = append(rows, []string{pos, strVal(c, "status_key"), strVal(c, "label"), terminal, transitions, instr})
	}
	cli.PrintTable(os.Stdout, headers, rows)
	return nil
}

func runPipelineColumnAdd(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if _, err := requireWorkspaceID(cmd); err != nil {
		return err
	}

	statusKey, _ := cmd.Flags().GetString("status-key")
	statusKey = strings.TrimSpace(statusKey)
	if statusKey == "" {
		return fmt.Errorf("--status-key is required")
	}
	if strings.ContainsAny(statusKey, " \t\n\r") {
		return fmt.Errorf("--status-key must not contain whitespace")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cols, err := fetchPipelineColumns(ctx, client, args[0])
	if err != nil {
		return err
	}

	for _, c := range cols {
		if strVal(c, "status_key") == statusKey {
			return fmt.Errorf("column with status_key %q already exists", statusKey)
		}
	}

	label, _ := cmd.Flags().GetString("label")
	if label == "" {
		label = statusKey
	}

	position, _ := cmd.Flags().GetInt("position")
	if position < 0 {
		// Append at end.
		maxPos := -1
		for _, c := range cols {
			if p, ok := c["position"].(float64); ok && int(p) > maxPos {
				maxPos = int(p)
			}
		}
		position = maxPos + 1
	}

	isTerminal, _ := cmd.Flags().GetBool("terminal")
	instructions, _ := cmd.Flags().GetString("instructions")
	allow, _ := cmd.Flags().GetStringSlice("allow")

	newCol := map[string]any{
		"status_key":          statusKey,
		"label":               label,
		"position":            position,
		"is_terminal":         isTerminal,
		"instructions":        instructions,
		"allowed_transitions": allow,
	}
	cols = append(cols, newCol)

	result, err := syncPipelineColumns(ctx, client, args[0], cols)
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, result)
	}
	fmt.Fprintf(os.Stdout, "Column %q added to pipeline %s.\n", statusKey, args[0])
	return nil
}

func runPipelineColumnUpdate(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if _, err := requireWorkspaceID(cmd); err != nil {
		return err
	}

	pipelineID := args[0]
	statusKey := args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cols, err := fetchPipelineColumns(ctx, client, pipelineID)
	if err != nil {
		return err
	}

	found := false
	for i, c := range cols {
		if strVal(c, "status_key") != statusKey {
			continue
		}
		found = true
		if v, _ := cmd.Flags().GetString("label"); cmd.Flags().Changed("label") {
			cols[i]["label"] = v
		}
		if v, _ := cmd.Flags().GetInt("position"); v >= 0 {
			cols[i]["position"] = v
		}
		if cmd.Flags().Changed("terminal") {
			cols[i]["is_terminal"] = true
		}
		if cmd.Flags().Changed("no-terminal") {
			cols[i]["is_terminal"] = false
		}
		if cmd.Flags().Changed("instructions") {
			v, _ := cmd.Flags().GetString("instructions")
			cols[i]["instructions"] = v
		}
		if cmd.Flags().Changed("allow") {
			v, _ := cmd.Flags().GetStringSlice("allow")
			cols[i]["allowed_transitions"] = v
		}
		break
	}
	if !found {
		return fmt.Errorf("column %q not found in pipeline %s", statusKey, pipelineID)
	}

	result, err := syncPipelineColumns(ctx, client, pipelineID, cols)
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, result)
	}
	fmt.Fprintf(os.Stdout, "Column %q updated.\n", statusKey)
	return nil
}

func runPipelineColumnDelete(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if _, err := requireWorkspaceID(cmd); err != nil {
		return err
	}

	pipelineID := args[0]
	statusKey := args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cols, err := fetchPipelineColumns(ctx, client, pipelineID)
	if err != nil {
		return err
	}

	filtered := cols[:0]
	for _, c := range cols {
		if strVal(c, "status_key") != statusKey {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == len(cols) {
		return fmt.Errorf("column %q not found in pipeline %s", statusKey, pipelineID)
	}

	if _, err := syncPipelineColumns(ctx, client, pipelineID, filtered); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Column %q deleted from pipeline %s.\n", statusKey, pipelineID)
	return nil
}

func runPipelineColumnSync(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}
	if _, err := requireWorkspaceID(cmd); err != nil {
		return err
	}

	var raw []byte
	if filePath, _ := cmd.Flags().GetString("file"); filePath != "" {
		raw, err = os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}
	} else {
		raw, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
	}

	var cols []map[string]any
	if err := json.Unmarshal(raw, &cols); err != nil {
		return fmt.Errorf("parse columns JSON: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := syncPipelineColumns(ctx, client, args[0], cols)
	if err != nil {
		return err
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, result)
	}
	fmt.Fprintf(os.Stdout, "Pipeline %s columns synced (%d columns).\n", args[0], len(result))
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func fetchPipelineColumns(ctx context.Context, client *cli.APIClient, pipelineID string) ([]map[string]any, error) {
	path := fmt.Sprintf("/api/workspaces/%s/pipelines/%s/columns", client.WorkspaceID, pipelineID)
	var cols []map[string]any
	if err := client.GetJSON(ctx, path, &cols); err != nil {
		return nil, fmt.Errorf("list pipeline columns: %w", err)
	}
	return cols, nil
}

func syncPipelineColumns(ctx context.Context, client *cli.APIClient, pipelineID string, cols []map[string]any) ([]map[string]any, error) {
	path := fmt.Sprintf("/api/workspaces/%s/pipelines/%s/columns", client.WorkspaceID, pipelineID)
	var result []map[string]any
	if err := client.PutJSON(ctx, path, cols, &result); err != nil {
		return nil, fmt.Errorf("sync pipeline columns: %w", err)
	}
	return result, nil
}

