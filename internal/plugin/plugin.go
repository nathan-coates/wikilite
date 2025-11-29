package plugin

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Plugin represents a loaded script ready for execution.
type Plugin struct {
	ID     string
	Script string
	Order  int
}

//go:embed types.d.ts
var typeDefinitionContent string

// loadFromDirectory scans a folder for plugins named in numerical order.
func loadFromDirectory(dir string) ([]Plugin, error) {
	err := ensureTypeDefinitions(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure types.d.ts: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	plugins := make([]Plugin, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".js" {
			continue
		}

		parts := strings.SplitN(entry.Name(), "-", 2)
		if len(parts) < 2 {
			continue
		}

		order, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		id := strings.TrimSuffix(parts[1], ".js")

		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read plugin %s: %w", entry.Name(), err)
		}

		plugins = append(plugins, Plugin{
			ID:     id,
			Order:  order,
			Script: string(content),
		})
	}

	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Order < plugins[j].Order
	})

	return plugins, nil
}

// ensureTypeDefinitions checks for the existence of types.d.ts and creates it if missing.
func ensureTypeDefinitions(dir string) error {
	path := filepath.Join(dir, "types.d.ts")

	_, err := os.Stat(path)
	if err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	err = os.WriteFile(path, []byte(typeDefinitionContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write types.d.ts: %w", err)
	}

	return nil
}
