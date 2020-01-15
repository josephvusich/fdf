package main

// #include <copyfile.h>
// #include <stdlib.h>
import "C"
import "unsafe"

func copyFile(src, dst string) error {
	csrc := C.CString(src)
	defer C.free(unsafe.Pointer(csrc))
	cdst := C.CString(dst)
	defer C.free(unsafe.Pointer(cdst))

	result, err := C.copyfile(csrc, cdst, nil, C.COPYFILE_DATA)
	if result != 0 {
		return err
	}
	return nil
}
