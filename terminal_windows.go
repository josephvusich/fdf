// +build windows

package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

func terminalANSI(enabled bool) (previous bool, err error) {
	handle := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return false, fmt.Errorf("unable to get console mode: %w", err)
	}

	previous = mode&windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0
	if previous == enabled {
		return previous, nil
	}

	if enabled {
		mode |= windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	} else {
		mode &= ^uint32(windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	}

	if err := windows.SetConsoleMode(handle, mode); err != nil {
		return false, fmt.Errorf("unable to set console mode: %w", err)
	}

	return previous, nil
}

func terminalWidth() (chars int, err error) {
	if chars, _, err = term.GetSize(int(os.Stdout.Fd())); err != nil {
		return -1, fmt.Errorf("unable to read info: %w", err)
	}
	return chars, nil
}
