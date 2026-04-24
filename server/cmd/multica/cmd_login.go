package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/multica-ai/multica/server/internal/cli"
)

// tryResolveAppURL returns the app URL if configured, or "" if not available.
// Unlike resolveAppURL, it never calls os.Exit.
func tryResolveAppURL(cmd *cobra.Command) string {
	for _, key := range []string{"MULTICA_APP_URL", "FRONTEND_ORIGIN"} {
		if val := strings.TrimSpace(os.Getenv(key)); val != "" {
			return strings.TrimRight(val, "/")
		}
	}
	profile := resolveProfile(cmd)
	cfg, err := cli.LoadCLIConfigForProfile(profile)
	if err == nil && cfg.AppURL != "" {
		return strings.TrimRight(cfg.AppURL, "/")
	}
	return ""
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate and set up workspaces",
	Long:  "Log in to Multica, then automatically discover and configure your workspaces.",
	RunE:  runLogin,
}

func init() {
	loginCmd.Flags().Bool("token", false, "Authenticate by pasting a personal access token")
}

func runLogin(cmd *cobra.Command, args []string) error {
	// Run the standard auth login flow.
	if err := runAuthLogin(cmd, args); err != nil {
		return err
	}

	// Auto-discover and watch all workspaces.
	if err := autoWatchWorkspaces(cmd); err != nil {
		fmt.Fprintf(os.Stderr, "\nCould not auto-configure workspaces: %v\n", err)
		fmt.Fprintf(os.Stderr, "Run 'multica workspace list' and 'multica workspace switch <id>' to set up manually.\n")
		return nil
	}

	fmt.Fprintf(os.Stderr, "\n→ Run 'multica daemon start' to start your local agent runtime.\n")
	return nil
}

func autoWatchWorkspaces(cmd *cobra.Command) error {
	serverURL := resolveServerURL(cmd)
	token := resolveToken(cmd)
	if token == "" {
		return fmt.Errorf("not authenticated")
	}

	client := cli.NewAPIClient(serverURL, "", token)
	workspaces, err := listWorkspaces(client)
	if err != nil {
		return fmt.Errorf("list workspaces: %w", err)
	}

	if len(workspaces) == 0 {
		workspaces, err = waitForWorkspaceCreation(cmd, client)
		if err != nil {
			return err
		}
		if len(workspaces) == 0 {
			fmt.Fprintln(os.Stderr, "\nNo workspaces found.")
			return nil
		}
	}

	profile := resolveProfile(cmd)
	cfg, err := cli.LoadCLIConfigForProfile(profile)
	if err != nil {
		return err
	}

	selected, err := chooseDefaultWorkspace(workspaces, cfg.WorkspaceID, os.Stdin, os.Stderr, stdinLooksInteractive())
	if err != nil {
		return err
	}
	cfg.WorkspaceID = selected.ID

	if err := cli.SaveCLIConfigForProfile(cfg, profile); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "\nFound %d workspace(s):\n", len(workspaces))
	for _, ws := range workspaces {
		marker := " "
		if ws.ID == selected.ID {
			marker = "*"
		}
		if ws.Slug != "" {
			fmt.Fprintf(os.Stderr, " %s %s [%s] (%s)\n", marker, ws.Name, ws.Slug, ws.ID)
			continue
		}
		fmt.Fprintf(os.Stderr, " %s %s (%s)\n", marker, ws.Name, ws.ID)
	}
	fmt.Fprintf(os.Stderr, "\nDefault workspace: %s (%s)\n", selected.Name, selected.ID)

	return nil
}

// waitForWorkspaceCreation opens the web workspace-creation page and polls
// until the user creates a workspace, returning the new workspace list.
func waitForWorkspaceCreation(cmd *cobra.Command, client *cli.APIClient) ([]workspaceSummary, error) {
	appURL := tryResolveAppURL(cmd)
	if appURL == "" {
		// No app URL available (e.g. token login without prior setup).
		// Can't open the browser — tell the user to create a workspace manually.
		fmt.Fprintln(os.Stderr, "\nNo workspaces found.")
		fmt.Fprintln(os.Stderr, "Create a workspace in the web dashboard, then run 'multica login' again.")
		return nil, nil
	}

	createWorkspaceURL := appURL + "/workspaces/new"

	fmt.Fprintln(os.Stderr, "\nNo workspaces found. Opening workspace creation in your browser...")
	if err := openBrowser(createWorkspaceURL); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open browser automatically.\n")
	}
	fmt.Fprintf(os.Stderr, "If the browser didn't open, visit:\n  %s\n", createWorkspaceURL)
	fmt.Fprintln(os.Stderr, "\nWaiting for workspace creation...")

	// Poll until a workspace appears or timeout (5 minutes).
	const pollInterval = 2 * time.Second
	const pollTimeout = 5 * time.Minute
	deadline := time.Now().Add(pollTimeout)

	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		var workspaces []workspaceSummary
		err := client.GetJSON(ctx, "/api/workspaces", &workspaces)
		cancel()

		if err != nil {
			continue // transient error, keep polling
		}
		if len(workspaces) > 0 {
			return workspaces, nil
		}
	}

	return nil, fmt.Errorf("timed out waiting for workspace creation")
}

func chooseDefaultWorkspace(workspaces []workspaceSummary, currentID string, in io.Reader, out io.Writer, interactive bool) (workspaceSummary, error) {
	if len(workspaces) == 0 {
		return workspaceSummary{}, fmt.Errorf("no workspaces available")
	}

	defaultIdx := 0
	for i, ws := range workspaces {
		if ws.ID == currentID {
			defaultIdx = i
			break
		}
	}

	if len(workspaces) == 1 || !interactive {
		return workspaces[defaultIdx], nil
	}

	fmt.Fprintln(out, "\nSelect the default workspace for this CLI profile:")
	for i, ws := range workspaces {
		currentMarker := ""
		if i == defaultIdx {
			currentMarker = " (default)"
		}
		if ws.Slug != "" {
			fmt.Fprintf(out, "  %d. %s [%s] (%s)%s\n", i+1, ws.Name, ws.Slug, ws.ID, currentMarker)
			continue
		}
		fmt.Fprintf(out, "  %d. %s (%s)%s\n", i+1, ws.Name, ws.ID, currentMarker)
	}

	reader := bufio.NewReader(in)
	for {
		fmt.Fprintf(out, "Enter number [default %d]: ", defaultIdx+1)
		answer, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return workspaceSummary{}, fmt.Errorf("read workspace selection: %w", err)
		}
		answer = strings.TrimSpace(answer)
		if answer == "" {
			return workspaces[defaultIdx], nil
		}

		n, convErr := strconv.Atoi(answer)
		if convErr == nil && n >= 1 && n <= len(workspaces) {
			return workspaces[n-1], nil
		}
		fmt.Fprintf(out, "Invalid selection %q. Enter a number between 1 and %d.\n", answer, len(workspaces))
	}
}

func stdinLooksInteractive() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
