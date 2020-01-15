package main

// #include <linux/fs.h>
import "C"

import (
	"os"

	"golang.org/x/sys/unix"
)

func cloneFile(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()
	df, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer df.Close()

	_, _, err = unix.Syscall(unix.SYS_IOCTL, df.Fd(), C.FICLONE, sf.Fd())
	return err
}
