package tray

import "fmt"

func restartAndQuit(restart func() error, quit func()) error {
	if restart == nil {
		return fmt.Errorf("restart is not configured")
	}
	if err := restart(); err != nil {
		return err
	}
	if quit != nil {
		quit()
	}
	return nil
}
