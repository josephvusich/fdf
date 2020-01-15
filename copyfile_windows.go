package main

// #define WIN32_LEAN_AND_MEAN
// #include <windows.h>
// #include <winbase.h>
import "C"

import (
	"syscall"
	"unsafe"
)

func copyFile(src, dst string) error {
	wsrc, err := syscall.UTF16PtrFromString(fixLongPath(src))
	if err != nil {
		return err
	}
	wdst, err := syscall.UTF16PtrFromString(fixLongPath(dst))
	if err != nil {
		return err
	}

	result, err := C.CopyFileW(C.PWCHAR(unsafe.Pointer(wsrc)), C.PWCHAR(unsafe.Pointer(wdst)), C.TRUE)
	if result == 0 {
		return err
	}
	return nil
}
