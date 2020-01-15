package main

// Adapted from https://github.com/git-lfs/git-lfs/blob/master/tools/util_windows.go
// Includes some bug fixes, such as file handles not being closed properly.

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	availableClusterSize = []int64{64 * 1024, 4 * 1024} // ReFS only supports 64KiB and 4KiB cluster.
	gigabyte             = int64(1024 * 1024 * 1024)
)

// Instructs the file system to copy a range of file bytes on behalf of an application.
//
// https://docs.microsoft.com/windows/win32/api/winioctl/ni-winioctl-fsctl_duplicate_extents_to_file
const fsctl_DUPLICATE_EXTENTS_TO_FILE = 623428

// Contains parameters for the FSCTL_DUPLICATE_EXTENTS control code that performs the Block Cloning operation.
//
// https://docs.microsoft.com/windows/win32/api/winioctl/ns-winioctl-duplicate_extents_data
type struct_DUPLICATE_EXTENTS_DATA struct {
	FileHandle       windows.Handle
	SourceFileOffset int64
	TargetFileOffset int64
	ByteCount        int64
}

// The source and destination regions must begin and end at a cluster boundary.
// The cloned region must be less than 4GB in length.
// The destination region must not extend past the end of file. If the application
//  wishes to extend the destination with cloned data, it must first call SetEndOfFile.
// https://docs.microsoft.com/en-us/windows/win32/fileio/block-cloning
func cloneFile(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()

	srcStat, err := sf.Stat()
	if err != nil {
		return err
	}

	df, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE, srcStat.Mode()) // No truncate version of os.Create
	if err != nil {
		return err
	}
	defer df.Close()

	fileSize := srcStat.Size()

	err = df.Truncate(fileSize) // set file size. There is a requirements "The destination region must not extend past the end of file."
	if err != nil {
		return err
	}

	offset := int64(0)

	// Requirement
	// * The source and destination regions must begin and end at a cluster boundary. (4KiB or 64KiB)
	// * cloneRegionSize less than 4GiB.
	// see https://docs.microsoft.com/windows/win32/fileio/block-cloning

	// Clone first xGiB region.
	for ; offset+gigabyte < fileSize; offset += gigabyte {
		err = callDuplicateExtentsToFile(df, sf, offset, gigabyte)
		if err != nil {
			return err
		}
	}

	// Clone tail. First try with 64KiB round up, then fallback to 4KiB.
	for _, cloneRegionSize := range availableClusterSize {
		err = callDuplicateExtentsToFile(df, sf, offset, roundUp(fileSize-offset, cloneRegionSize))
		if err != nil {
			continue
		}
		break
	}

	return err
}

// call FSCTL_DUPLICATE_EXTENTS_TO_FILE IOCTL
// see https://docs.microsoft.com/en-us/windows/win32/api/winioctl/ni-winioctl-fsctl_duplicate_extents_to_file
//
// memo: Overflow (cloneRegionSize is greater than file ends) is safe and just ignored by windows.
func callDuplicateExtentsToFile(dst, src *os.File, offset int64, cloneRegionSize int64) (err error) {
	var (
		bytesReturned uint32
		overlapped    windows.Overlapped
	)

	request := struct_DUPLICATE_EXTENTS_DATA{
		FileHandle:       windows.Handle(src.Fd()),
		SourceFileOffset: offset,
		TargetFileOffset: offset,
		ByteCount:        cloneRegionSize,
	}

	return windows.DeviceIoControl(
		windows.Handle(dst.Fd()),
		fsctl_DUPLICATE_EXTENTS_TO_FILE,
		(*byte)(unsafe.Pointer(&request)),
		uint32(unsafe.Sizeof(request)),
		(*byte)(unsafe.Pointer(nil)), // = nullptr
		0,
		&bytesReturned,
		&overlapped)
}

func roundUp(value, base int64) int64 {
	mod := value % base
	if mod == 0 {
		return value
	}

	return value - mod + base
}
