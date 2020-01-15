package main

// #include <sys/clonefile.h>
// #include <stdlib.h>
import "C"
import "unsafe"

func cloneFile(src, dst string) error {
	csrc := C.CString(src)
	defer C.free(unsafe.Pointer(csrc))
	cdst := C.CString(dst)
	defer C.free(unsafe.Pointer(cdst))

	result, err := C.clonefile(csrc, cdst, 0)
	if result != 0 {
		return err
	}
	return nil
}
