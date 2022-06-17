
## Copy-on-write Cloning

The `--clone` flag enables copy-on-write clones on compatible filesystems. Common filesystems with support include APFS, ReFS, and Btrfs. See [Comparison of file systems](https://en.wikipedia.org/wiki/Comparison_of_file_systems) on Wikipedia for more. Note that `--copy` may also create clones when using Mac OS X with an APFS filesystem.

## License

Licensed under the [Apache 2.0 license](LICENSE).

* [clonefile_windows.go](clonefile_windows.go) is adapted from [git-lfs](https://github.com/git-lfs/git-lfs/blob/285eebdddf3a47e83d3cc457397b2bcc798cf935/tools/util_windows.go), licensed under the [MIT license](LICENSE-git-lfs.md).
* [path_windows.go](path_windows.go) is adapted from the [Go standard library](https://github.com/golang/go/blob/b86e76681366447798c94abb959bb60875bcc856/src/os/path_windows.go), licensed under a [BSD-style license](LICENSE-golang).
