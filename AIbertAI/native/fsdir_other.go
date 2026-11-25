//go:build !windows

package main

import "os"

func syncDirFile(f *os.File) error         { return f.Sync() }
func privateDirMode(info os.FileInfo) bool { return info.Mode().Perm()&0077 == 0 }
