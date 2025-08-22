//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

const ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004

var (
	stdOutHandle = uintptr(^uint32(10) + 1) // -11
	stdErrHandle = uintptr(^uint32(11) + 1) // -12
)

func enableVT(which uintptr) {
	k32 := syscall.NewLazyDLL("kernel32.dll")
	pGetStdHandle := k32.NewProc("GetStdHandle")
	pGetConsoleMode := k32.NewProc("GetConsoleMode")
	pSetConsoleMode := k32.NewProc("SetConsoleMode")

	h, _, _ := pGetStdHandle.Call(which)
	if h == 0 || h == uintptr(syscall.InvalidHandle) {
		return
	}
	var mode uint32
	r1, _, _ := pGetConsoleMode.Call(h, uintptr(unsafe.Pointer(&mode)))
	if r1 == 0 {
		return
	}
	_, _, _ = pSetConsoleMode.Call(h, uintptr(mode|ENABLE_VIRTUAL_TERMINAL_PROCESSING))
}

func init() {
	enableVT(stdOutHandle)
	enableVT(stdErrHandle)
}
