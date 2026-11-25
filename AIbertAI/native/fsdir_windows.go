//go:build windows

package main

import "os"

// Windows has no POSIX permission bits (directories report 0777) and
// FlushFileBuffers on a directory handle fails, so NTFS metadata journaling
// stands in for directory fsync and the profile ACL for 0700.
func syncDirFile(*os.File) error      { return nil }
func privateDirMode(os.FileInfo) bool { return true }
