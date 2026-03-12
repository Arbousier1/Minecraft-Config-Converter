//go:build !cgo

package desktopui

import "errors"

func Run(baseDir string) error {
	return errors.New("desktop UI requires cgo-enabled build environment")
}
