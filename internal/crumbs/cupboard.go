// Package crumbs provides a convenience wrapper around the crumbs Cupboard.
// Implements: prd-cupboard-core (via crumbs module);
//
//	docs/ARCHITECTURE § Cupboard Integration, § Crumbs Client.
//
// This wrapper initializes the SQLite backend, attaches the cupboard, and
// provides typed accessor methods for crumb operations. Callers use
// NewCupboard to create an instance and Close to release resources.
package crumbs

import (
	"fmt"

	"github.com/petar-djukic/crumbs/pkg/sqlite"
	"github.com/petar-djukic/crumbs/pkg/types"
)

// Default data directory for crumbs storage.
const DefaultDataDir = ".crumbs"

// Error wrapping for cobbler context.
var (
	ErrCupboardInit   = fmt.Errorf("cobbler: cupboard initialization failed")
	ErrCupboardAttach = fmt.Errorf("cobbler: cupboard attach failed")
	ErrTableAccess    = fmt.Errorf("cobbler: table access failed")
	ErrCrumbGet       = fmt.Errorf("cobbler: crumb get failed")
	ErrCrumbSet       = fmt.Errorf("cobbler: crumb set failed")
	ErrCrumbFetch     = fmt.Errorf("cobbler: crumb fetch failed")
)

// Cupboard wraps the crumbs Cupboard interface with typed convenience methods.
// It initializes an SQLite backend, attaches the cupboard, and provides
// direct access to crumb operations without re-abstracting the interface.
type Cupboard struct {
	backend types.Cupboard
	dataDir string
}

// NewCupboard creates a new Cupboard wrapper using SQLite backend.
// The dataDir parameter specifies where to store the SQLite database;
// if empty, defaults to DefaultDataDir (.crumbs).
// Returns an error if backend creation or attach fails.
func NewCupboard(dataDir string) (*Cupboard, error) {
	if dataDir == "" {
		dataDir = DefaultDataDir
	}

	backend := sqlite.NewBackend()

	config := types.Config{
		Backend: types.BackendSQLite,
		DataDir: dataDir,
	}

	if err := backend.Attach(config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCupboardAttach, err)
	}

	return &Cupboard{
		backend: backend,
		dataDir: dataDir,
	}, nil
}

// Close detaches the cupboard and releases all resources.
// After Close, all operations will fail. Close is idempotent.
func (c *Cupboard) Close() error {
	if c.backend == nil {
		return nil
	}
	return c.backend.Detach()
}

// GetCrumb retrieves a crumb by ID from the crumbs table.
// Returns the typed Crumb or an error if not found or access fails.
func (c *Cupboard) GetCrumb(id string) (*types.Crumb, error) {
	table, err := c.backend.GetTable(types.CrumbsTable)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTableAccess, err)
	}

	entity, err := table.Get(id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCrumbGet, err)
	}

	crumb, ok := entity.(*types.Crumb)
	if !ok {
		return nil, fmt.Errorf("%w: unexpected type %T", ErrCrumbGet, entity)
	}

	return crumb, nil
}

// SetCrumb creates or updates a crumb in the crumbs table.
// If id is empty, a new UUID v7 is generated.
// Returns the actual ID (generated or provided) or an error.
func (c *Cupboard) SetCrumb(id string, crumb *types.Crumb) (string, error) {
	table, err := c.backend.GetTable(types.CrumbsTable)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTableAccess, err)
	}

	actualID, err := table.Set(id, crumb)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCrumbSet, err)
	}

	return actualID, nil
}

// FetchCrumbs queries crumbs matching the filter.
// Filter keys are field names; values are required field values.
// An empty filter returns all crumbs.
// Returns typed Crumb slices or an error.
func (c *Cupboard) FetchCrumbs(filter map[string]any) ([]*types.Crumb, error) {
	table, err := c.backend.GetTable(types.CrumbsTable)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTableAccess, err)
	}

	entities, err := table.Fetch(filter)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCrumbFetch, err)
	}

	crumbs := make([]*types.Crumb, 0, len(entities))
	for _, entity := range entities {
		crumb, ok := entity.(*types.Crumb)
		if !ok {
			return nil, fmt.Errorf("%w: unexpected type %T in results", ErrCrumbFetch, entity)
		}
		crumbs = append(crumbs, crumb)
	}

	return crumbs, nil
}

// GetTable provides direct access to a table by name.
// Use this for operations beyond crumb convenience methods.
func (c *Cupboard) GetTable(name string) (types.Table, error) {
	return c.backend.GetTable(name)
}

// DataDir returns the data directory used by this cupboard.
func (c *Cupboard) DataDir() string {
	return c.dataDir
}
