package main

import (
	"syscall"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

const (
	LWA_ALPHA    = 0x00000002
	LWA_COLORKEY = 0x00000001
)

var (
	libuser32 *windows.LazyDLL

	setLayeredWindowAttributes *windows.LazyProc
)

func init() {

	// Libary
	libuser32 = windows.NewLazySystemDLL("user32.dll")

	// Functions
	setLayeredWindowAttributes = libuser32.NewProc("SetLayeredWindowAttributes")
}

func SetLayeredWindowAttributes(hwnd win.HWND, crKey win.COLORREF, bAlpha byte, dwFlags uint32) bool {

	ret, _, _ := syscall.Syscall6(setLayeredWindowAttributes.Addr(), 4,
		uintptr(hwnd),
		uintptr(crKey),
		uintptr(bAlpha),
		uintptr(dwFlags),
		0,
		0)

	return ret != 0
}
