package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func lockDesktopSessionStore(root *os.Root) (*os.File, error) {
	lock, err := root.OpenFile("session-writer.lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	var offset windows.Overlapped
	if err := windows.LockFileEx(windows.Handle(lock.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &offset); err != nil {
		lock.Close()
		return nil, fmt.Errorf("员工端会话目录已被占用: %w", err)
	}
	return lock, nil
}
