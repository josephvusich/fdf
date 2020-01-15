# File Duplicate Finder (fdf)

A cross-platform duplicate file finder supporting deduplication via copy-on-write clones and hard links. Inspired by [Olof Laderkvist's Windows-only fdf utility](http://www.ltr-data.se/opencode.html/).

## System Requirements

* Go 1.13+ with CGO support enabled
* One of the following platforms:

| Platform | Minimum version | Reason |
|---|---|---|
| Linux | 2.6.33+ | File-to-file `sendfile()` support |
| Mac OS X | Sierra 10.12+ | `clonefile()` and APFS support |
| Windows | Windows 10 or Windows Server 2016+ | `FSCTL_DUPLICATE_EXTENTS_TO_FILE` support |

## Copy-on-write Cloning

Pass the `--clone` flag to enable copy-on-write clones on compatible filesystems. Common filesystems with support include APFS, ReFS, Btrfs, and ZFS. See [Comparison of file systems](https://en.wikipedia.org/wiki/Comparison_of_file_systems) on Wikipedia for more. Note that `--copy` may also create clones when using Mac OS X with an APFS filesystem.

## License

Licensed under the [Apache 2.0 license](LICENSE).

* [clonefile_windows.go](clonefile_windows.go) is adapted from [git-lfs](https://github.com/git-lfs/git-lfs/blob/285eebdddf3a47e83d3cc457397b2bcc798cf935/tools/util_windows.go), licensed under the [MIT license](LICENSE-git-lfs.md).
* [path_windows.go](path_windows.go) is adapted from the [Go standard library](https://github.com/golang/go/blob/b86e76681366447798c94abb959bb60875bcc856/src/os/path_windows.go), licensed under a [BSD-style license](LICENSE-golang).
