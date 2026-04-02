package main

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

	return unix.IoctlFileClone(int(df.Fd()), int(sf.Fd()))
}
