package main

// #include <sys/clonefile.h>
// #include <stdlib.h>
// #include <errno.h>
//
// int cloneAndFree(char *src, char *dst) {
//   int result = clonefile(src, dst, 0);
//   int err = errno;
//   free(src);
//   free(dst);
//   errno = err;
//   return result;
// }
import "C"

func cloneFile(src, dst string) error {
	result, err := C.cloneAndFree(C.CString(src), C.CString(dst))
	if result != 0 {
		return err
	}
	return nil
}
