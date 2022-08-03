# File Duplicate Finder (fdf)

![build status](https://github.com/josephvusich/fdf/actions/workflows/go.yml/badge.svg?branch=master)

A cross-platform duplicate file finder supporting deduplication via copy-on-write clones and hard links. Inspired by [Olof Laderkvist's Windows-only fdf utility](http://www.ltr-data.se/opencode.html/).

## Installation
### System Requirements

* Go (with CGO support enabled on non-Windows platforms)
* One of the following platforms:

| Platform | Minimum version | Reason |
|---|---|---|
| Linux | 2.6.33+ | File-to-file `sendfile()` support |
| Mac OS X | Sierra 10.12+ | `clonefile()` and APFS support |
| Windows | Windows 10 or Windows Server 2016+ | `FSCTL_DUPLICATE_EXTENTS_TO_FILE` support |

Clone this repository with `go get -u` and then `go install`.

## Usage
