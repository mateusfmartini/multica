package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/cli"
)

func TestSlugifyWorkspaceName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"My Team", "my-team"},
		{"  Product   Ops  ", "product-ops"},
		{"Acme_2026!", "acme-2026"},
		{"日本語", ""},
	}

	for _, tt := range tests {
		if got := slugifyWorkspaceName(tt.in); got != tt.want {
			t.Fatalf("slugifyWorkspaceName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRunWorkspaceCreateUsesExistingAPI(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/workspaces" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "ws-new",
			"name": gotBody["name"],
			"slug": gotBody["slug"],
		})
	}))
	defer srv.Close()

	t.Setenv("MULTICA_SERVER_URL", srv.URL)
	t.Setenv("MULTICA_TOKEN", "test-token")

	cmd := testCmd()
	cmd.Flags().String("slug", "", "")
	cmd.Flags().String("description", "", "")
	cmd.Flags().String("context", "", "")
	cmd.Flags().String("issue-prefix", "", "")
	cmd.Flags().String("output", "json", "")

	out := captureStdout(t, func() {
		if err := runWorkspaceCreate(cmd, []string{"My Team"}); err != nil {
			t.Fatalf("runWorkspaceCreate() error = %v", err)
		}
	})

	if gotBody["name"] != "My Team" {
		t.Fatalf("body name = %#v, want %q", gotBody["name"], "My Team")
	}
	if gotBody["slug"] != "my-team" {
		t.Fatalf("body slug = %#v, want %q", gotBody["slug"], "my-team")
	}
	if !strings.Contains(out, `"id": "ws-new"`) {
		t.Fatalf("stdout = %q, want created workspace JSON", out)
	}
}

func TestRunWorkspaceSwitchValidatesAndPersistsConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MULTICA_TOKEN", "test-token")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/workspaces/ws-2" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "ws-2",
			"name": "Beta",
			"slug": "beta",
		})
	}))
	defer srv.Close()

	t.Setenv("MULTICA_SERVER_URL", srv.URL)

	cmd := testCmd()
	stderr := captureStderr(t, func() {
		if err := runWorkspaceSwitch(cmd, []string{"ws-2"}); err != nil {
			t.Fatalf("runWorkspaceSwitch() error = %v", err)
		}
	})

	cfg, err := cli.LoadCLIConfig()
	if err != nil {
		t.Fatalf("LoadCLIConfig() error = %v", err)
	}
	if cfg.WorkspaceID != "ws-2" {
		t.Fatalf("WorkspaceID = %q, want %q", cfg.WorkspaceID, "ws-2")
	}
	if !strings.Contains(stderr, "Switched active workspace to Beta (ws-2).") {
		t.Fatalf("stderr = %q, want switch confirmation", stderr)
	}
}

func TestRunWorkspaceListShowsSlugAndCurrentMarker(t *testing.T) {
	t.Setenv("MULTICA_TOKEN", "test-token")
	t.Setenv("MULTICA_WORKSPACE_ID", "ws-2")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/workspaces" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": "ws-1", "name": "Alpha", "slug": "alpha"},
			{"id": "ws-2", "name": "Beta", "slug": "beta"},
		})
	}))
	defer srv.Close()

	t.Setenv("MULTICA_SERVER_URL", srv.URL)

	cmd := testCmd()
	cmd.Flags().String("output", "table", "")

	out := captureStdout(t, func() {
		if err := runWorkspaceList(cmd, nil); err != nil {
			t.Fatalf("runWorkspaceList() error = %v", err)
		}
	})

	if !strings.Contains(out, "SLUG") {
		t.Fatalf("stdout = %q, want SLUG header", out)
	}
	if !strings.Contains(out, "beta") {
		t.Fatalf("stdout = %q, want workspace slug", out)
	}
	if !strings.Contains(out, "ws-2") || !strings.Contains(out, "Beta") || !strings.Contains(out, "*") {
		t.Fatalf("stdout = %q, want current workspace marker", out)
	}
}

func TestChooseDefaultWorkspacePromptsAndUsesSelection(t *testing.T) {
	workspaces := []workspaceSummary{
		{ID: "ws-1", Name: "Alpha", Slug: "alpha"},
		{ID: "ws-2", Name: "Beta", Slug: "beta"},
	}

	var out bytes.Buffer
	selected, err := chooseDefaultWorkspace(workspaces, "ws-1", strings.NewReader("2\n"), &out, true)
	if err != nil {
		t.Fatalf("chooseDefaultWorkspace() error = %v", err)
	}
	if selected.ID != "ws-2" {
		t.Fatalf("selected ID = %q, want %q", selected.ID, "ws-2")
	}
	if !strings.Contains(out.String(), "Enter number [default 1]:") {
		t.Fatalf("prompt output = %q, want default prompt", out.String())
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	return captureFile(t, &os.Stdout, fn)
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	return captureFile(t, &os.Stderr, fn)
}

func captureFile(t *testing.T, target **os.File, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe(): %v", err)
	}
	orig := *target
	*target = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()

	_ = w.Close()
	*target = orig
	return <-done
}
