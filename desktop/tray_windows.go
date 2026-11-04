//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procRegisterClassExW   = user32.NewProc("RegisterClassExW")
	procCreateWindowExW    = user32.NewProc("CreateWindowExW")
	procDefWindowProcW     = user32.NewProc("DefWindowProcW")
	procGetMessageW        = user32.NewProc("GetMessageW")
	procTranslateMessage   = user32.NewProc("TranslateMessage")
	procDispatchMessageW   = user32.NewProc("DispatchMessageW")
	procLoadIconW          = user32.NewProc("LoadIconW")
	procPostQuitMessage    = user32.NewProc("PostQuitMessage")
	procShellNotifyIconW   = shell32.NewProc("Shell_NotifyIconW")
	procGetModuleHandleW   = kernel32.NewProc("GetModuleHandleW")
)

const (
	WM_CLOSE      = 0x0010
	WM_DESTROY    = 0x0002
	WM_TRAYICON   = 0x0400 + 1
	WM_RBUTTONUP  = 0x0205
	WM_LBUTTONUP  = 0x0202

	NIM_ADD     = 0x00000000
	NIM_DELETE  = 0x00000002
	NIF_ICON    = 0x00000002
	NIF_MESSAGE = 0x00000001
	NIF_TIP     = 0x00000004

	IDI_APPLICATION = 32512

	HWND_MESSAGE = ^uintptr(2) // -3
)

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type point struct{ X, Y int32 }

type msg struct {
	HWnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
}

var trayHwnd uintptr

func trayWndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	if message == WM_TRAYICON {
		// no-op; icon is status only
	}
	if message == WM_DESTROY {
		procPostQuitMessage.Call(0)
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}

func runTray() {
	className, _ := syscall.UTF16PtrFromString("BatonTrayClass")
	hInstance, _, _ := procGetModuleHandleW.Call(0)

	wndProc := syscall.NewCallback(trayWndProc)
	wc := wndClassEx{
		CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		LpfnWndProc:   wndProc,
		HInstance:     hInstance,
		LpszClassName: className,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	title, _ := syscall.UTF16PtrFromString("Baton Tray")
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		0, 0, 0, 0, 0,
		HWND_MESSAGE, 0, hInstance, 0,
	)
	if hwnd == 0 {
		return
	}
	trayHwnd = hwnd

	hIcon, _, _ := procLoadIconW.Call(0, IDI_APPLICATION)

	nid := notifyIconData{
		CbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:             hwnd,
		UID:              1,
		UFlags:           NIF_ICON | NIF_MESSAGE | NIF_TIP,
		UCallbackMessage: WM_TRAYICON,
		HIcon:            hIcon,
	}
	tip, _ := syscall.UTF16FromString("Baton audio handoff")
	copy(nid.SzTip[:], tip)

	procShellNotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&nid)))

	// Message loop
	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if r == 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}

	// Cleanup
	procShellNotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
}