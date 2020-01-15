package main

// #include <sys/sendfile.h>
import "C"

import "os"

func copyFile(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()

	st, err := sf.Stat()
	if err != nil {
		return err
	}

	df, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer df.Close()

	n, err := C.sendfile(C.int(df.Fd()), C.int(sf.Fd()), nil, C.ulong(st.Size()))
	if n < 0 {
		return err
	}
	return nil
}
