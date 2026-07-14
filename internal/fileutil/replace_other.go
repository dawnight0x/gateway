//go:build !windows

package fileutil

import "os"

func replaceFile(source, target string) error {
	return os.Rename(source, target)
}
