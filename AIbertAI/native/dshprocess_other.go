//go:build !windows

package main

import "os"

// Elsewhere DSH's children get their own stdio, so killing DSH is enough.
func dshProcessTree(*os.Process) func() { return func() {} }
