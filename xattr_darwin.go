package main

import (
	"errors"

	"golang.org/x/sys/unix"
)

const xattrKey = "com.josephvusich.fdf.hash"

func xattrNotFound(err error) bool {
	return errors.Is(err, unix.ENOATTR)
}

func getXattr(path, name string) ([]byte, error) {
	size, err := unix.Getxattr(path, name, nil)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, size)
	_, err = unix.Getxattr(path, name, buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func setXattr(path, name string, data []byte) error {
	return unix.Setxattr(path, name, data, 0)
}
