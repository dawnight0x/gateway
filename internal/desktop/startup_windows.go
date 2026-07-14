package desktop

import (
	"errors"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const startupRunKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func IsAutostartEnabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, startupRunKey, registry.QUERY_VALUE)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	defer key.Close()

	value, _, err := key.GetStringValue(startupAppName)
	if err != nil {
		if errors.Is(err, registry.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	want, err := startupCommand(currentExecutable())
	if err != nil {
		return false, err
	}
	return strings.EqualFold(value, want), nil
}

func SetAutostartEnabled(enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, startupRunKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if !enabled {
		if err := key.DeleteValue(startupAppName); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		return nil
	}

	cmd, err := startupCommand(currentExecutable())
	if err != nil {
		return err
	}
	return key.SetStringValue(startupAppName, cmd)
}
