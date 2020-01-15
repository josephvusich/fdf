// +build windows

package main

// #define WIN32_LEAN_AND_MEAN
// #include <windows.h>
// #include <wincon.h>
//
// #ifndef ENABLE_VIRTUAL_TERMINAL_PROCESSING
// #define ENABLE_VIRTUAL_TERMINAL_PROCESSING 0x0004
// #endif
//
import "C"

import "errors"

func terminalANSI(enabled bool) (previous bool, err error) {
	handle := C.GetStdHandle(C.STD_OUTPUT_HANDLE)
	if handle == C.HANDLE(C.INVALID_HANDLE_VALUE) {
		return false, errors.New("unable to get STDOUT handle")
	}

	var mode C.DWORD
	if C.GetConsoleMode(handle, &mode) == 0 {
		return false, errors.New("unable to get console mode")
	}

	previous = mode&C.ENABLE_VIRTUAL_TERMINAL_PROCESSING != 0
	if previous == enabled {
		return previous, nil
	}

	if enabled {
		mode |= C.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	} else {
		mode &= ^C.DWORD(C.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	}

	if C.SetConsoleMode(handle, mode) == 0 {
		return false, errors.New("unable to set console mode")
	}

	return previous, nil
}

func terminalWidth() (chars int, err error) {
	hStdOut := C.GetStdHandle(C.STD_OUTPUT_HANDLE)
	var info C.CONSOLE_SCREEN_BUFFER_INFO
	if C.GetConsoleScreenBufferInfo(hStdOut, &info) == 0 {
		return -1, errors.New("unable to read info")
	}
	return int(info.dwSize.X), nil
}
