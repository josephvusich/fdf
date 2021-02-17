# File Duplicate Finder (fdf)

A cross-platform duplicate file finder supporting deduplication via copy-on-write clones and hard links. Inspired by [Olof Laderkvist's Windows-only fdf utility](http://www.ltr-data.se/opencode.html/).

## Installation
### System Requirements

* Go 1.13+ with CGO support enabled
* One of the following platforms:

| Platform | Minimum version | Reason |
|---|---|---|
| Linux | 2.6.33+ | File-to-file `sendfile()` support |
| Mac OS X | Sierra 10.12+ | `clonefile()` and APFS support |
| Windows | Windows 10 or Windows Server 2016+ | `FSCTL_DUPLICATE_EXTENTS_TO_FILE` support |

Clone this repository with `go get -u` and then `go install`.

## Usage
```
fdf [-a | -c | -d | -l] [-thqr] [-m FIELDS] [-z BYTES] [directory ...]
  -a, --clone               (verb) create copy-on-write clones instead of hardlinks (not supported on all filesystems)
  -c, --copy                (verb) split existing hardlinks via copy
                            mutually exclusive with --ignore-hardlinks
  -d, --delete              (verb) delete duplicate files
  -t, --dry-run             don't actually do anything, just show what would be done
      --help                show this help screen and exit
  -h, --ignore-hardlinks    ignore existing hardlinks
                            mutually exclusive with --copy
  -l, --link                (verb) hardlink duplicate files
  -m, --match FIELDS        Evaluate FIELDS to determine file equality, where valid fields are:
                              name, or name[offset:len,offset:len,...] (case insensitive)
                                [0:-1] whole string
                                [0:-2] all except last character
                                 [1:2] second and third characters
                                [-1:1] last character
                                [-3:3] last 3 characters
                              copyname (e.g., 'foo.bar' == 'foo (1).bar' == 'Copy of foo.bar', must specify +size or +content)
                              size
                              content (default, also implies size)
                            specify multiple fields using '+', e.g.: name+content
  -z, --minimum-size BYTES  skip files smaller than BYTES (default 1)
      --preserve PATTERN    (deprecated) alias for --protect PATTERN
  -p, --protect PATTERN     prevent files matching glob PATTERN from being modified or deleted
                            may appear more than once to support multiple patterns
                            rules are applied in the order specified
  -q, --quiet               don't display current filename during scanning
  -r, --recursive           traverse subdirectories
  -n, --skip-header LENGTH  skip LENGTH bytes at the beginning of each file when comparing
                            implies --minimum-size LENGTH+1
      --unprotect value     remove files added by --protect
                            may appear more than once
                            rules are applied in the order specified
  -v, --verbose             display additional details regarding protected paths
```

## Copy-on-write Cloning

The `--clone` flag enables copy-on-write clones on compatible filesystems. Common filesystems with support include APFS, ReFS, and Btrfs. See [Comparison of file systems](https://en.wikipedia.org/wiki/Comparison_of_file_systems) on Wikipedia for more. Note that `--copy` may also create clones when using Mac OS X with an APFS filesystem.

## License

Licensed under the [Apache 2.0 license](LICENSE).

* [clonefile_windows.go](clonefile_windows.go) is adapted from [git-lfs](https://github.com/git-lfs/git-lfs/blob/285eebdddf3a47e83d3cc457397b2bcc798cf935/tools/util_windows.go), licensed under the [MIT license](LICENSE-git-lfs.md).
* [path_windows.go](path_windows.go) is adapted from the [Go standard library](https://github.com/golang/go/blob/b86e76681366447798c94abb959bb60875bcc856/src/os/path_windows.go), licensed under a [BSD-style license](LICENSE-golang).
