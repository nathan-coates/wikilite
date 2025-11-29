//go:build !plugins

package plugin

type Manager struct{}

// NewManager is a placeholder function for when the plugin system is not built.
func NewManager(_ string, _ string, _ string) (*Manager, error) {
	return nil, nil
}

func (*Manager) Close() error {
	return nil
}
