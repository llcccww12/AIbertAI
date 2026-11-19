//go:build !windows

package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func lockDesktopSessionStore(root *os.Root) (*os.File, error) {
	lock, err := root.OpenFile("session-writer.lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		lock.Close()
		return nil, fmt.Errorf("员工端会话目录已被占用: %w", err)
	}
	return lock, nil
}
