package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/multica-ai/multica/server/internal/cli"
)

var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Work with workspaces",
}

var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all workspaces you belong to",
	RunE:  runWorkspaceList,
}

var workspaceCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new workspace",
	Args:  exactArgs(1),
	RunE:  runWorkspaceCreate,
}

var workspaceSwitchCmd = &cobra.Command{
	Use:   "switch <workspace-id>",
	Short: "Switch the active workspace in the current profile",
	Args:  exactArgs(1),
	RunE:  runWorkspaceSwitch,
}

var workspaceGetCmd = &cobra.Command{
	Use:   "get [workspace-id]",
	Short: "Get workspace details",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runWorkspaceGet,
}

var workspaceMembersCmd = &cobra.Command{
	Use:   "members [workspace-id]",
	Short: "List workspace members",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runWorkspaceMembers,
}

func init() {
	workspaceCmd.AddCommand(workspaceListCmd)
	workspaceCmd.AddCommand(workspaceCreateCmd)
	workspaceCmd.AddCommand(workspaceSwitchCmd)
	workspaceCmd.AddCommand(workspaceGetCmd)
	workspaceCmd.AddCommand(workspaceMembersCmd)

	workspaceListCmd.Flags().String("output", "table", "Output format: table or json")
	workspaceCreateCmd.Flags().String("slug", "", "Workspace slug (defaults to a slugified version of the name)")
	workspaceCreateCmd.Flags().String("description", "", "Workspace description")
	workspaceCreateCmd.Flags().String("context", "", "Workspace context")
	workspaceCreateCmd.Flags().String("issue-prefix", "", "Issue prefix override")
	workspaceCreateCmd.Flags().String("output", "json", "Output format: table or json")
	workspaceGetCmd.Flags().String("output", "json", "Output format: table or json")
	workspaceMembersCmd.Flags().String("output", "table", "Output format: table or json")
}

type workspaceSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func listWorkspaces(client *cli.APIClient) ([]workspaceSummary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var workspaces []workspaceSummary
	if err := client.GetJSON(ctx, "/api/workspaces", &workspaces); err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	return workspaces, nil
}

func runWorkspaceList(cmd *cobra.Command, _ []string) error {
	serverURL := resolveServerURL(cmd)
	token := resolveToken(cmd)
	if token == "" {
		return fmt.Errorf("not authenticated: run 'multica login' first")
	}

	client := cli.NewAPIClient(serverURL, "", token)
	workspaces, err := listWorkspaces(client)
	if err != nil {
		return err
	}

	if len(workspaces) == 0 {
		fmt.Fprintln(os.Stderr, "No workspaces found.")
		return nil
	}

	currentID := resolveWorkspaceID(cmd)
	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		resp := make([]map[string]any, 0, len(workspaces))
		for _, ws := range workspaces {
			resp = append(resp, map[string]any{
				"id":      ws.ID,
				"name":    ws.Name,
				"slug":    ws.Slug,
				"current": ws.ID == currentID,
			})
		}
		return cli.PrintJSON(os.Stdout, resp)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSLUG\tCURRENT")
	for _, ws := range workspaces {
		current := ""
		if ws.ID == currentID {
			current = "*"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", ws.ID, ws.Name, ws.Slug, current)
	}
	return w.Flush()
}

func runWorkspaceCreate(cmd *cobra.Command, args []string) error {
	serverURL := resolveServerURL(cmd)
	token := resolveToken(cmd)
	if token == "" {
		return fmt.Errorf("not authenticated: run 'multica login' first")
	}

	name := args[0]
	slug, _ := cmd.Flags().GetString("slug")
	if slug == "" {
		slug = slugifyWorkspaceName(name)
	}
	if slug == "" {
		return fmt.Errorf("could not derive a valid slug from %q; pass --slug explicitly", name)
	}

	body := map[string]any{
		"name": name,
		"slug": slug,
	}
	if v, _ := cmd.Flags().GetString("description"); v != "" {
		body["description"] = v
	}
	if v, _ := cmd.Flags().GetString("context"); v != "" {
		body["context"] = v
	}
	if v, _ := cmd.Flags().GetString("issue-prefix"); v != "" {
		body["issue_prefix"] = v
	}

	client := cli.NewAPIClient(serverURL, "", token)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var result map[string]any
	if err := client.PostJSON(ctx, "/api/workspaces", body, &result); err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "table" {
		headers := []string{"ID", "NAME", "SLUG"}
		rows := [][]string{{
			strVal(result, "id"),
			strVal(result, "name"),
			strVal(result, "slug"),
		}}
		cli.PrintTable(os.Stdout, headers, rows)
		return nil
	}

	return cli.PrintJSON(os.Stdout, result)
}

func runWorkspaceSwitch(cmd *cobra.Command, args []string) error {
	serverURL := resolveServerURL(cmd)
	token := resolveToken(cmd)
	if token == "" {
		return fmt.Errorf("not authenticated: run 'multica login' first")
	}

	client := cli.NewAPIClient(serverURL, "", token)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var ws map[string]any
	if err := client.GetJSON(ctx, "/api/workspaces/"+args[0], &ws); err != nil {
		return fmt.Errorf("get workspace: %w", err)
	}

	profile := resolveProfile(cmd)
	cfg, err := cli.LoadCLIConfigForProfile(profile)
	if err != nil {
		return err
	}
	cfg.WorkspaceID = strVal(ws, "id")
	if err := cli.SaveCLIConfigForProfile(cfg, profile); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Switched active workspace to %s (%s).\n", strVal(ws, "name"), strVal(ws, "id"))
	return nil
}

func workspaceIDFromArgs(cmd *cobra.Command, args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return resolveWorkspaceID(cmd)
}

func runWorkspaceGet(cmd *cobra.Command, args []string) error {
	wsID := workspaceIDFromArgs(cmd, args)
	if wsID == "" {
		return fmt.Errorf("workspace ID is required: pass as argument or set MULTICA_WORKSPACE_ID")
	}

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var ws map[string]any
	if err := client.GetJSON(ctx, "/api/workspaces/"+wsID, &ws); err != nil {
		return fmt.Errorf("get workspace: %w", err)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "table" {
		desc := strVal(ws, "description")
		if utf8.RuneCountInString(desc) > 60 {
			runes := []rune(desc)
			desc = string(runes[:57]) + "..."
		}
		wsContext := strVal(ws, "context")
		if utf8.RuneCountInString(wsContext) > 60 {
			runes := []rune(wsContext)
			wsContext = string(runes[:57]) + "..."
		}
		headers := []string{"ID", "NAME", "SLUG", "DESCRIPTION", "CONTEXT"}
		rows := [][]string{{
			strVal(ws, "id"),
			strVal(ws, "name"),
			strVal(ws, "slug"),
			desc,
			wsContext,
		}}
		cli.PrintTable(os.Stdout, headers, rows)
		return nil
	}

	return cli.PrintJSON(os.Stdout, ws)
}

func runWorkspaceMembers(cmd *cobra.Command, args []string) error {
	wsID := workspaceIDFromArgs(cmd, args)
	if wsID == "" {
		return fmt.Errorf("workspace ID is required: pass as argument or set MULTICA_WORKSPACE_ID")
	}

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var members []map[string]any
	if err := client.GetJSON(ctx, "/api/workspaces/"+wsID+"/members", &members); err != nil {
		return fmt.Errorf("list members: %w", err)
	}

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		return cli.PrintJSON(os.Stdout, members)
	}

	headers := []string{"USER ID", "NAME", "EMAIL", "ROLE"}
	rows := make([][]string, 0, len(members))
	for _, m := range members {
		rows = append(rows, []string{
			strVal(m, "user_id"),
			strVal(m, "name"),
			strVal(m, "email"),
			strVal(m, "role"),
		})
	}
	cli.PrintTable(os.Stdout, headers, rows)
	return nil
}

func slugifyWorkspaceName(name string) string {
	var b []rune
	lastHyphen := true
	for _, r := range name {
		switch {
		case unicode.IsLetter(r):
			lower := unicode.ToLower(r)
			if lower >= 'a' && lower <= 'z' {
				b = append(b, lower)
				lastHyphen = false
			} else if !lastHyphen {
				b = append(b, '-')
				lastHyphen = true
			}
		case unicode.IsDigit(r):
			b = append(b, r)
			lastHyphen = false
		default:
			if !lastHyphen {
				b = append(b, '-')
				lastHyphen = true
			}
		}
	}
	slug := string(b)
	for len(slug) > 0 && slug[len(slug)-1] == '-' {
		slug = slug[:len(slug)-1]
	}
	return slug
}
