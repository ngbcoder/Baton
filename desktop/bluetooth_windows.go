//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/ngbcoder/baton/desktop/protocol"
)

var (
	bthprops = syscall.NewLazyDLL("bthprops.cpl")
	procFindFirstDevice  = bthprops.NewProc("BluetoothFindFirstDevice")
	procFindNextDevice   = bthprops.NewProc("BluetoothFindNextDevice")
	procFindDeviceClose  = bthprops.NewProc("BluetoothFindDeviceClose")
	procGetDeviceInfo    = bthprops.NewProc("BluetoothGetDeviceInfo")
	procSetServiceState  = bthprops.NewProc("BluetoothSetServiceState")
	procFindFirstRadio   = bthprops.NewProc("BluetoothFindFirstRadio")
	procFindRadioClose   = bthprops.NewProc("BluetoothFindRadioClose")
)

type btAddress struct{ ullLong uint64 }

type bluetoothDeviceInfo struct {
	dwSize             uint32
	address            btAddress
	ulClassofDevice    uint32
	fConnected         bool
	fRemembered        bool
	fAuthenticated     bool
	stLastSeen         [8]byte
	stLastUsed         [8]byte
	szName             [248]uint16
}

type bluetoothDeviceSearchParams struct {
	dwSize               uint32
	fReturnAuthenticated uint32
	fReturnRemembered    uint32
	fReturnUnknown       uint32
	fReturnConnected     uint32
	fIssueInquiry        uint32
	cTimeoutMultiplier   byte
	hRadio               uintptr
}

type bluetoothFindRadioParams struct {
	dwSize uint32
}

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var audioSinkGUID = guid{
	Data1: 0x0000110b,
	Data2: 0x0000,
	Data3: 0x1000,
	Data4: [8]byte{0x80, 0x00, 0x00, 0x80, 0x5f, 0x9b, 0x34, 0xfb},
}

func formatBtAddr(ull uint64) string {
	return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
		(ull>>40)&0xff, (ull>>32)&0xff, (ull>>24)&0xff,
		(ull>>16)&0xff, (ull>>8)&0xff, ull&0xff)
}

func parseBtAddr(s string) uint64 {
	var b [6]uint64
	// accept colon or dash separated hex
	fmt.Sscanf(s, "%x:%x:%x:%x:%x:%x", &b[0], &b[1], &b[2], &b[3], &b[4], &b[5])
	// also try dash
	if b[0] == 0 && b[1] == 0 && b[2] == 0 && b[3] == 0 && b[4] == 0 && b[5] == 0 {
		fmt.Sscanf(s, "%x-%x-%x-%x-%x-%x", &b[0], &b[1], &b[2], &b[3], &b[4], &b[5])
	}
	return (b[0] << 40) | (b[1] << 32) | (b[2] << 24) | (b[3] << 16) | (b[4] << 8) | b[5]
}

func EnumeratePairedDevices() ([]string, []string, error) {
	params := bluetoothDeviceSearchParams{
		dwSize:               uint32(unsafe.Sizeof(bluetoothDeviceSearchParams{})),
		fReturnAuthenticated: 1,
		fReturnRemembered:    1,
		fReturnConnected:     1,
	}
	var names, addrs []string
	di := bluetoothDeviceInfo{dwSize: uint32(unsafe.Sizeof(bluetoothDeviceInfo{}))}
	hFind, _, _ := procFindFirstDevice.Call(uintptr(unsafe.Pointer(&params)), uintptr(unsafe.Pointer(&di)))
	if hFind == 0 {
		return names, addrs, nil
	}
	defer procFindDeviceClose.Call(hFind)
	for {
		names = append(names, syscall.UTF16ToString(di.szName[:]))
		addrs = append(addrs, formatBtAddr(di.address.ullLong))
		ret, _, _ := procFindNextDevice.Call(hFind, uintptr(unsafe.Pointer(&di)))
		if ret == 0 {
			break
		}
	}
	return names, addrs, nil
}

func getDeviceInfo(addr string) (bluetoothDeviceInfo, bool) {
	di := bluetoothDeviceInfo{dwSize: uint32(unsafe.Sizeof(bluetoothDeviceInfo{}))}
	di.address.ullLong = parseBtAddr(addr)
	ret, _, _ := procGetDeviceInfo.Call(0, uintptr(unsafe.Pointer(&di)))
	if ret != 0 { // ERROR_SUCCESS == 0
		return di, false
	}
	return di, true
}

func findFirstRadio() uintptr {
	params := bluetoothFindRadioParams{dwSize: uint32(unsafe.Sizeof(bluetoothFindRadioParams{}))}
	var hRadio uintptr
	hFind, _, _ := procFindFirstRadio.Call(uintptr(unsafe.Pointer(&params)), uintptr(unsafe.Pointer(&hRadio)))
	if hFind != 0 {
		procFindRadioClose.Call(hFind)
	}
	return hRadio
}

// windowsBluetooth implements protocol.BluetoothController using Win32 APIs.
type windowsBluetooth struct{}

func NewWindowsBluetooth() *windowsBluetooth { return &windowsBluetooth{} }

func defaultBluetooth() protocol.BluetoothController { return NewWindowsBluetooth() }

func (w *windowsBluetooth) PairedDevices() ([]protocol.BtDevice, error) {
	names, addrs, err := EnumeratePairedDevices()
	if err != nil {
		return nil, err
	}
	var out []protocol.BtDevice
	for i := range names {
		out = append(out, protocol.BtDevice{Address: addrs[i], Name: names[i]})
	}
	return out, nil
}

func (w *windowsBluetooth) IsConnected(addr string) (bool, error) {
	di, ok := getDeviceInfo(addr)
	if !ok {
		return false, fmt.Errorf("device not found")
	}
	return di.fConnected, nil
}

func (w *windowsBluetooth) Connect(addr string) error {
	hRadio := findFirstRadio()
	if hRadio == 0 {
		return fmt.Errorf("no radio")
	}
	defer syscall.CloseHandle(syscall.Handle(hRadio))
	di, ok := getDeviceInfo(addr)
	if !ok {
		return fmt.Errorf("device not found")
	}
	ret, _, _ := procSetServiceState.Call(hRadio, uintptr(unsafe.Pointer(&di)), uintptr(unsafe.Pointer(&audioSinkGUID)), 1)
	if ret != 0 {
		return fmt.Errorf("connect failed: %d", ret)
	}
	return nil
}

func (w *windowsBluetooth) Disconnect(addr string) error {
	hRadio := findFirstRadio()
	if hRadio == 0 {
		return fmt.Errorf("no radio")
	}
	defer syscall.CloseHandle(syscall.Handle(hRadio))
	di, ok := getDeviceInfo(addr)
	if !ok {
		return fmt.Errorf("device not found")
	}
	ret, _, _ := procSetServiceState.Call(hRadio, uintptr(unsafe.Pointer(&di)), uintptr(unsafe.Pointer(&audioSinkGUID)), 0)
	if ret != 0 {
		return fmt.Errorf("disconnect failed: %d", ret)
	}
	return nil
}

func (w *windowsBluetooth) IsMultipointConnected(addr string) (bool, error) {
	return false, nil
}