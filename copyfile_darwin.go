package main

// #include <copyfile.h>
// #include <stdlib.h>
// #include <errno.h>
//
// int copyAndFree(char *src, char *dst) {
//   int result = copyfile(src, dst, NULL, COPYFILE_CLONE);
//   int err = errno;
//   free(src);
//   free(dst);
//   errno = err;
//   return result;
// }
import "C"

func copyFile(src, dst string) error {
	result, err := C.copyAndFree(C.CString(src), C.CString(dst))
	if result != 0 {
		return err
	}
	return nil
}
