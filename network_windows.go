//go:build windows

package main

import (
	"encoding/binary"
	"syscall"
	"unsafe"
)

var getIfTable = syscall.NewLazyDLL("iphlpapi.dll").NewProc("GetIfTable")

const (
	ifRowSize        = 860 // MIB_IFROW
	ifRowTypeOffset  = 516
	ifRowStatusOff   = 544
	ifRowInOctetsOff = 552
	ifTypeLoopback   = 24
	ifOperational    = 5
)

// systemReceivedBytes returns a running total of bytes received by all operational, non-loopback network
// adapters. It is a 32-bit counter per adapter, so callers should only use the difference between two
// readings taken a short time apart.
func systemReceivedBytes() (uint32, bool) {
	size := uint32(0)
	getIfTable.Call(0, uintptr(unsafe.Pointer(&size)), 0)
	if size == 0 || size > 1<<22 {
		return 0, false
	}
	buffer := make([]byte, size)
	if result, _, _ := getIfTable.Call(uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&size)), 0); result != 0 {
		return 0, false
	}
	count := int(binary.LittleEndian.Uint32(buffer))
	var total uint32
	for index := 0; index < count; index++ {
		start := 4 + index*ifRowSize
		if start+ifRowSize > len(buffer) {
			break
		}
		row := buffer[start : start+ifRowSize]
		if binary.LittleEndian.Uint32(row[ifRowTypeOffset:]) == ifTypeLoopback || binary.LittleEndian.Uint32(row[ifRowStatusOff:]) != ifOperational {
			continue
		}
		total += binary.LittleEndian.Uint32(row[ifRowInOctetsOff:])
	}
	return total, true
}
