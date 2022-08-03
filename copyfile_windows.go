package main

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	libKernel32   = windows.NewLazySystemDLL("kernel32.dll")
	procCopyFileW = libKernel32.NewProc("CopyFileW")
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

	result, _, err := syscall.SyscallN(procCopyFileW.Addr(), uintptr(unsafe.Pointer(wsrc)), uintptr(unsafe.Pointer(wdst)), uintptr(1))
	if result == 0 {
		return err
	}
	return nil
}
