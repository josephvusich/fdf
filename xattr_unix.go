// +build !windows,!darwin

package main

import (
	"golang.org/x/sys/unix"
)

// Linux requires the "user." namespace prefix for user-space extended attributes.
const xattrKey = "user.com.josephvusich.fdf.hash"

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
