// +build windows

package main

import "errors"

var errXattrNotSupported = errors.New("xattr not supported on Windows")

func getXattr(path, name string) ([]byte, error) {
	return nil, errXattrNotSupported
}

func setXattr(path, name string, data []byte) error {
	return errXattrNotSupported
}
