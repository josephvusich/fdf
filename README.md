# File Duplicate Finder (fdf)

![build status](https://github.com/josephvusich/fdf/actions/workflows/go.yml/badge.svg?branch=master)

A cross-platform duplicate file finder supporting deduplication via copy-on-write clones and hard links. Inspired by [Olof Laderkvist's Windows-only fdf utility](http://www.ltr-data.se/opencode.html/).

## Installation

`go install github.com/josephvusich/fdf@latest`

### System Requirements

* Go (with CGO support enabled on non-Windows platforms)
* One of the following platforms:

| Platform | Minimum version | Reason |
|---|---|---|
| Linux | 2.6.33+ | File-to-file `sendfile()` support |
| Mac OS X | Sierra 10.12+ | `clonefile()` and APFS support |
| Windows | Windows 10 or Windows Server 2016+ | `FSCTL_DUPLICATE_EXTENTS_TO_FILE` support |

## Usage
```
usage: fdf [--clone | --copy | --delete | --link] [-hqrtv]
        [-m FIELDS] [-z BYTES] [-n LENGTH]
        [--protect PATTERN] [--unprotect PATTERN] [directory ...]

  -a, --clone               (verb) create copy-on-write clones instead of hardlinks (not supported on all filesystems)
  -c, --copy                (verb) split existing hardlinks via copy
                            mutually exclusive with --ignore-hardlinks
  -d, --delete              (verb) delete duplicate files
  -t, --dry-run             don't actually do anything, just show what would be done
      --exclude GLOB        exclude files matching GLOB from scanning
      --exclude-dir DIR     exclude DIR from scanning, throws error if DIR does not exist
      --help                show this help screen and exit
      --ignore-content      allow --match without 'content'
  -h, --ignore-hardlinks    ignore existing hardlinks
                            mutually exclusive with --copy
      --include GLOB        include GLOB, opposite of --exclude
      --include-dir DIR     include DIR, throws error if DIR does not exist
  -l, --link                (verb) hardlink duplicate files
  -m, --match FIELDS        Evaluate FIELDS to determine file equality, where valid fields are:
                              name (case insensitive)
                                range notation supported: name[offset:len,offset:len,...]
                                  name[0:-1] whole string
                                  name[0:-2] all except last character
                                  name[1:2]  second and third characters
                                  name[-1:1] last character
                                  name[-3:3] last 3 characters
                              copyname (case insensitive)
                                'foo.bar' == 'foo (1).bar' == 'Copy of foo.bar', also requires +size or +content
                              namesuffix (case insensitive)
                                one filename must end with the other, e.g.: 'foo-1.bar' and '1.bar'
                              parent (case insensitive name of immediate parent directory)
                                range notation supported: see 'name' for examples
                              path
                                match parent directory path
                              relpath
                                match parent directory path relative to input dir(s)
                              size
                              content (default, also implies size)
                            specify multiple fields using '+', e.g.: name+content
  -z, --minimum-size BYTES  skip files smaller than BYTES, must be greater than the sum of --skip-header and --skip-footer (default 1)
      --preserve PATTERN    (deprecated) alias for --protect PATTERN
  -p, --protect PATTERN     prevent files matching glob PATTERN from being modified or deleted
                            may appear more than once to support multiple patterns
                            rules are applied in the order specified
      --protect-dir DIR     similar to --protect 'DIR/**/*', but throws error if DIR does not exist
  -q, --quiet               don't display current filename during scanning
  -r, --recursive           traverse subdirectories
      --skip-footer LENGTH  skip LENGTH bytes at the end of each file when comparing
  -n, --skip-header LENGTH  skip LENGTH bytes at the beginning of each file when comparing
      --unprotect value     remove files added by --protect
                            may appear more than once
                            rules are applied in the order specified
      --unprotect-dir DIR   similar to --unprotect 'DIR/**/*', but throws error if DIR does not exist
  -v, --verbose             display additional details regarding protected paths
```

## Copy-on-write Cloning

The `--clone` flag enables copy-on-write clones on compatible filesystems. Common filesystems with support include APFS, ReFS, and Btrfs. See [Comparison of file systems](https://en.wikipedia.org/wiki/Comparison_of_file_systems) on Wikipedia for more. Note that `--copy` may also create clones when using Mac OS X with an APFS filesystem.

## License

Licensed under the [Apache 2.0 license](LICENSE).

* [clonefile_windows.go](clonefile_windows.go) is adapted from [git-lfs](https://github.com/git-lfs/git-lfs/blob/285eebdddf3a47e83d3cc457397b2bcc798cf935/tools/util_windows.go), licensed under the [MIT license](LICENSE-git-lfs.md).
* [path_windows.go](path_windows.go) is adapted from the [Go standard library](https://github.com/golang/go/blob/b86e76681366447798c94abb959bb60875bcc856/src/os/path_windows.go), licensed under a [BSD-style license](LICENSE-golang).
