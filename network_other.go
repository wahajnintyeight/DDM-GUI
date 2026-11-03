//go:build !windows

package main

func systemReceivedBytes() (uint32, bool) { return 0, false }
