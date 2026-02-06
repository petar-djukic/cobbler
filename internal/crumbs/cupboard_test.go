package crumbs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/petar-djukic/crumbs/pkg/types"
)

// tempDir creates a temporary directory for test data.
func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "cobbler-crumbs-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

func TestNewCupboard(t *testing.T) {
	dataDir := tempDir(t)

	cupboard, err := NewCupboard(dataDir)
	if err != nil {
		t.Fatalf("NewCupboard failed: %v", err)
	}
	defer cupboard.Close()

	// Verify dataDir is set correctly
	if cupboard.DataDir() != dataDir {
		t.Errorf("DataDir = %q, want %q", cupboard.DataDir(), dataDir)
	}

	// Verify database file exists
	dbPath := filepath.Join(dataDir, "cupboard.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("database file not created at %s", dbPath)
	}
}

func TestNewCupboard_DefaultDataDir(t *testing.T) {
	// Test with empty dataDir uses default
	tmpDir := tempDir(t)
	originalWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer os.Chdir(originalWd)

	cupboard, err := NewCupboard("")
	if err != nil {
		t.Fatalf("NewCupboard with empty dataDir failed: %v", err)
	}
	defer cupboard.Close()

	if cupboard.DataDir() != DefaultDataDir {
		t.Errorf("DataDir = %q, want %q", cupboard.DataDir(), DefaultDataDir)
	}
}

func TestSetCrumb_Create(t *testing.T) {
	dataDir := tempDir(t)

	cupboard, err := NewCupboard(dataDir)
	if err != nil {
		t.Fatalf("NewCupboard failed: %v", err)
	}
	defer cupboard.Close()

	crumb := &types.Crumb{
		Name:  "Test Crumb",
		State: types.StateReady,
	}

	// Create with empty ID generates new UUID
	id, err := cupboard.SetCrumb("", crumb)
	if err != nil {
		t.Fatalf("SetCrumb failed: %v", err)
	}

	if id == "" {
		t.Error("SetCrumb returned empty ID")
	}

	// Verify crumb was created
	retrieved, err := cupboard.GetCrumb(id)
	if err != nil {
		t.Fatalf("GetCrumb failed: %v", err)
	}

	if retrieved.Name != crumb.Name {
		t.Errorf("Name = %q, want %q", retrieved.Name, crumb.Name)
	}
	if retrieved.State != crumb.State {
		t.Errorf("State = %q, want %q", retrieved.State, crumb.State)
	}
}

func TestSetCrumb_Update(t *testing.T) {
	dataDir := tempDir(t)

	cupboard, err := NewCupboard(dataDir)
	if err != nil {
		t.Fatalf("NewCupboard failed: %v", err)
	}
	defer cupboard.Close()

	// Create initial crumb
	crumb := &types.Crumb{
		Name:  "Initial Name",
		State: types.StateReady,
	}
	id, err := cupboard.SetCrumb("", crumb)
	if err != nil {
		t.Fatalf("SetCrumb (create) failed: %v", err)
	}

	// Update the crumb
	crumb.Name = "Updated Name"
	crumb.State = types.StateTaken
	_, err = cupboard.SetCrumb(id, crumb)
	if err != nil {
		t.Fatalf("SetCrumb (update) failed: %v", err)
	}

	// Verify update
	retrieved, err := cupboard.GetCrumb(id)
	if err != nil {
		t.Fatalf("GetCrumb failed: %v", err)
	}

	if retrieved.Name != "Updated Name" {
		t.Errorf("Name = %q, want %q", retrieved.Name, "Updated Name")
	}
	if retrieved.State != types.StateTaken {
		t.Errorf("State = %q, want %q", retrieved.State, types.StateTaken)
	}
}

func TestGetCrumb_NotFound(t *testing.T) {
	dataDir := tempDir(t)

	cupboard, err := NewCupboard(dataDir)
	if err != nil {
		t.Fatalf("NewCupboard failed: %v", err)
	}
	defer cupboard.Close()

	_, err = cupboard.GetCrumb("nonexistent-id")
	if err == nil {
		t.Error("GetCrumb with nonexistent ID should return error")
	}
}

func TestFetchCrumbs_All(t *testing.T) {
	dataDir := tempDir(t)

	cupboard, err := NewCupboard(dataDir)
	if err != nil {
		t.Fatalf("NewCupboard failed: %v", err)
	}
	defer cupboard.Close()

	// Create multiple crumbs
	crumbs := []*types.Crumb{
		{Name: "Crumb 1", State: types.StateReady},
		{Name: "Crumb 2", State: types.StateReady},
		{Name: "Crumb 3", State: types.StateTaken},
	}

	for _, c := range crumbs {
		if _, err := cupboard.SetCrumb("", c); err != nil {
			t.Fatalf("SetCrumb failed: %v", err)
		}
	}

	// Fetch all
	results, err := cupboard.FetchCrumbs(nil)
	if err != nil {
		t.Fatalf("FetchCrumbs failed: %v", err)
	}

	if len(results) != len(crumbs) {
		t.Errorf("FetchCrumbs returned %d crumbs, want %d", len(results), len(crumbs))
	}
}

func TestFetchCrumbs_Filtered(t *testing.T) {
	dataDir := tempDir(t)

	cupboard, err := NewCupboard(dataDir)
	if err != nil {
		t.Fatalf("NewCupboard failed: %v", err)
	}
	defer cupboard.Close()

	// Create crumbs with different states
	crumbData := []*types.Crumb{
		{Name: "Ready 1", State: types.StateReady},
		{Name: "Ready 2", State: types.StateReady},
		{Name: "Taken 1", State: types.StateTaken},
	}

	for _, c := range crumbData {
		if _, err := cupboard.SetCrumb("", c); err != nil {
			t.Fatalf("SetCrumb failed: %v", err)
		}
	}

	// Fetch only ready crumbs (use struct field name "State" as filter key)
	filter := map[string]any{"State": types.StateReady}
	results, err := cupboard.FetchCrumbs(filter)
	if err != nil {
		t.Fatalf("FetchCrumbs failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("FetchCrumbs(state=ready) returned %d crumbs, want 2", len(results))
	}

	for _, c := range results {
		if c.State != types.StateReady {
			t.Errorf("FetchCrumbs returned crumb with State = %q, want %q", c.State, types.StateReady)
		}
	}
}

func TestClose(t *testing.T) {
	dataDir := tempDir(t)

	cupboard, err := NewCupboard(dataDir)
	if err != nil {
		t.Fatalf("NewCupboard failed: %v", err)
	}

	// Close should succeed
	if err := cupboard.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Close again should be idempotent (backend.Detach is idempotent)
	if err := cupboard.Close(); err != nil {
		t.Errorf("second Close failed: %v", err)
	}
}

func TestClose_OperationsFailAfterClose(t *testing.T) {
	dataDir := tempDir(t)

	cupboard, err := NewCupboard(dataDir)
	if err != nil {
		t.Fatalf("NewCupboard failed: %v", err)
	}

	cupboard.Close()

	// Operations should fail after close
	_, err = cupboard.GetCrumb("any-id")
	if err == nil {
		t.Error("GetCrumb after Close should return error")
	}

	_, err = cupboard.SetCrumb("", &types.Crumb{Name: "Test", State: types.StateReady})
	if err == nil {
		t.Error("SetCrumb after Close should return error")
	}

	_, err = cupboard.FetchCrumbs(nil)
	if err == nil {
		t.Error("FetchCrumbs after Close should return error")
	}
}

func TestGetTable(t *testing.T) {
	dataDir := tempDir(t)

	cupboard, err := NewCupboard(dataDir)
	if err != nil {
		t.Fatalf("NewCupboard failed: %v", err)
	}
	defer cupboard.Close()

	// GetTable should return valid table
	table, err := cupboard.GetTable(types.CrumbsTable)
	if err != nil {
		t.Fatalf("GetTable failed: %v", err)
	}

	if table == nil {
		t.Error("GetTable returned nil")
	}
}

func TestGetTable_NotFound(t *testing.T) {
	dataDir := tempDir(t)

	cupboard, err := NewCupboard(dataDir)
	if err != nil {
		t.Fatalf("NewCupboard failed: %v", err)
	}
	defer cupboard.Close()

	_, err = cupboard.GetTable("nonexistent-table")
	if err == nil {
		t.Error("GetTable with nonexistent table should return error")
	}
}

func TestCrumbWithProperties(t *testing.T) {
	// Note: The crumbs backend currently stores properties in memory during Set
	// but does not load them back during Get. Properties are stored separately
	// in the crumb_properties table and require the Properties system to be
	// used properly (defining Property entities first).
	//
	// This test verifies that crumbs with Properties set can be stored and
	// retrieved, even though the Properties map won't be populated on Get.
	dataDir := tempDir(t)

	cupboard, err := NewCupboard(dataDir)
	if err != nil {
		t.Fatalf("NewCupboard failed: %v", err)
	}
	defer cupboard.Close()

	crumb := &types.Crumb{
		Name:  "Crumb with Properties",
		State: types.StateReady,
		Properties: map[string]any{
			"work_type":   "documentation",
			"priority":    1,
			"description": "Test description",
		},
	}

	id, err := cupboard.SetCrumb("", crumb)
	if err != nil {
		t.Fatalf("SetCrumb failed: %v", err)
	}

	// Verify crumb can be retrieved (properties not loaded by Get)
	retrieved, err := cupboard.GetCrumb(id)
	if err != nil {
		t.Fatalf("GetCrumb failed: %v", err)
	}

	if retrieved.Name != crumb.Name {
		t.Errorf("Name = %q, want %q", retrieved.Name, crumb.Name)
	}
	if retrieved.CrumbID != id {
		t.Errorf("CrumbID = %q, want %q", retrieved.CrumbID, id)
	}
}
