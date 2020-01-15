// +build !windows

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

func terminalANSI(enabled bool) (previous bool, err error) {
	return true, nil
}

func terminalWidth() (chars int, err error) {
	size, err := unix.IoctlGetWinsize(int(os.Stdout.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		return -1, err
	}
	return int(size.Col), nil
}
