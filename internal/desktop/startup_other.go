//go:build !windows

package desktop

import "fmt"

func IsAutostartEnabled() (bool, error) {
	return false, nil
}

func SetAutostartEnabled(enabled bool) error {
	if !enabled {
		return nil
	}
	return fmt.Errorf("autostart is only implemented on windows")
}
