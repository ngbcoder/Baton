//go:build windows

package main

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/ngbcoder/baton/desktop/protocol"
)

// COM helpers
var (
	clsidMMDeviceEnumerator = windows.GUID{Data1: 0xBCDE0395, Data2: 0xE52F, Data3: 0x467C, Data4: [8]byte{0x8E, 0x3D, 0xC4, 0x57, 0x92, 0x91, 0x69, 0x2E}}
	iidIMMDeviceEnumerator  = windows.GUID{Data1: 0xA95664D2, Data2: 0x9614, Data3: 0x4F35, Data4: [8]byte{0xA7, 0x46, 0xDE, 0x8D, 0xB6, 0x36, 0x17, 0xE6}}
	clsidPolicyConfig        = windows.GUID{Data1: 0x870AF99C, Data2: 0x171D, Data3: 0x4F9E, Data4: [8]byte{0xAF, 0x0D, 0xE6, 0x3D, 0xF4, 0x0C, 0x2B, 0xC9}}
	iidIPolicyConfig         = windows.GUID{Data1: 0xF8679F50, Data2: 0x850A, Data3: 0x41CF, Data4: [8]byte{0x9C, 0x72, 0x43, 0x0F, 0x29, 0x02, 0x90, 0xC8}}
)

const (
	CLSCTX_ALL          = 23
	COINIT_APARTMENT    = 2
	STGM_READ           = 0
	DEVICE_STATE_ACTIVE = 1
	E_RENDER            = 0
)

type iUnknownVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
}

type immDeviceVtbl struct {
	iUnknownVtbl
	Activate                      uintptr
	OpenPropertyStore             uintptr
	GetId                         uintptr
	GetState                      uintptr
}

type immDevice struct{ lpVtbl *immDeviceVtbl }

type immDeviceCollectionVtbl struct {
	iUnknownVtbl
	GetCount uintptr
	Item     uintptr
}

type immDeviceCollection struct{ lpVtbl *immDeviceCollectionVtbl }

type immDeviceEnumeratorVtbl struct {
	iUnknownVtbl
	EnumAudioEndpoints               uintptr
	GetDefaultAudioEndpoint          uintptr
	GetDevice                       uintptr
	RegisterEndpointNotificationCallback uintptr
	UnregisterEndpointNotificationCallback uintptr
}

type immDeviceEnumerator struct{ lpVtbl *immDeviceEnumeratorVtbl }

type iPropertyStoreVtbl struct {
	iUnknownVtbl
	GetCount  uintptr
	GetAt     uintptr
	GetValue  uintptr
	SetValue  uintptr
	Commit    uintptr
}

type iPropertyStore struct{ lpVtbl *iPropertyStoreVtbl }

type policyConfigVtbl struct {
	iUnknownVtbl
	GetMixFormat         uintptr
	GetDeviceFormat      uintptr
	ResetDeviceFormat    uintptr
	SetDeviceFormat      uintptr
	GetProcessingPeriod  uintptr
	SetProcessingPeriod  uintptr
	GetShareMode         uintptr
	SetShareMode         uintptr
	GetPropertyValue     uintptr
	SetPropertyValue     uintptr
	SetDefaultEndpoint   uintptr
	SetEndpointVisibility uintptr
}

type policyConfig struct{ lpVtbl *policyConfigVtbl }

type propertyKey struct {
	FmtID windows.GUID
	Pid   uint32
}

type propVariant struct {
	Vt         uint16
	Reserved   [6]byte
	PointerVal uintptr
}

var pkeyFriendlyName = propertyKey{
	FmtID: windows.GUID{Data1: 0xA45C254E, Data2: 0xDF1C, Data3: 0x4EFD, Data4: [8]byte{0x80, 0x20, 0x67, 0xD1, 0x46, 0xA8, 0x50, 0xE0}},
	Pid:   14,
}

type windowsAudio struct{}

func defaultAudio() protocol.AudioController { return &windowsAudio{} }

func (a *windowsAudio) PausePlayback() error {
	// Send media play/pause key.
	cmd := "powershell -NoProfile -Command \"$wsh = New-Object -ComObject WScript.Shell; $wsh.SendKeys([char]0xB3)\""
	return runCmd(cmd)
}

func (a *windowsAudio) trySetDefaultOutput(target string) error {
	_ = windows.CoInitializeEx(0, COINIT_APARTMENT)
	defer windows.CoUninitialize()

	target = strings.ToLower(target)

	// Create device enumerator.
	var enumerator *immDeviceEnumerator
	hr := coCreateInstance(&clsidMMDeviceEnumerator, &iidIMMDeviceEnumerator, unsafe.Pointer(&enumerator))
	if hr != 0 {
		return fmt.Errorf("CoCreateInstance enumerator failed: 0x%x", hr)
	}
	defer syscall.SyscallN(enumerator.lpVtbl.Release, uintptr(unsafe.Pointer(enumerator)))

	// Enumerate active render endpoints.
	var collection *immDeviceCollection
	r1, _, _ := syscall.SyscallN(enumerator.lpVtbl.EnumAudioEndpoints, uintptr(unsafe.Pointer(enumerator)), E_RENDER, DEVICE_STATE_ACTIVE, uintptr(unsafe.Pointer(&collection)))
	if r1 != 0 {
		return fmt.Errorf("EnumAudioEndpoints failed: 0x%x", r1)
	}
	defer syscall.SyscallN(collection.lpVtbl.Release, uintptr(unsafe.Pointer(collection)))

	var count uint32
	syscall.SyscallN(collection.lpVtbl.GetCount, uintptr(unsafe.Pointer(collection)), uintptr(unsafe.Pointer(&count)))

	var foundID string
	for i := uint32(0); i < count; i++ {
		var dev *immDevice
		syscall.SyscallN(collection.lpVtbl.Item, uintptr(unsafe.Pointer(collection)), uintptr(i), uintptr(unsafe.Pointer(&dev)))
		if dev == nil {
			continue
		}
		// Get id
		var idPtr *uint16
		syscall.SyscallN(dev.lpVtbl.GetId, uintptr(unsafe.Pointer(dev)), uintptr(unsafe.Pointer(&idPtr)))
		id := windows.UTF16PtrToString(idPtr)
		windows.CoTaskMemFree(unsafe.Pointer(idPtr))

		// Open property store
		var props *iPropertyStore
		syscall.SyscallN(dev.lpVtbl.OpenPropertyStore, uintptr(unsafe.Pointer(dev)), STGM_READ, uintptr(unsafe.Pointer(&props)))
		if props != nil {
			var pv propVariant
			syscall.SyscallN(props.lpVtbl.GetValue, uintptr(unsafe.Pointer(props)), uintptr(unsafe.Pointer(&pkeyFriendlyName)), uintptr(unsafe.Pointer(&pv)))
			if pv.PointerVal != 0 {
				name := windows.UTF16PtrToString((*uint16)(unsafe.Pointer(pv.PointerVal)))
				if strings.Contains(strings.ToLower(name), target) {
					foundID = id
				}
			}
			syscall.SyscallN(props.lpVtbl.Release, uintptr(unsafe.Pointer(props)))
		}
		syscall.SyscallN(dev.lpVtbl.Release, uintptr(unsafe.Pointer(dev)))
		if foundID != "" {
			break
		}
	}

	if foundID == "" {
		return fmt.Errorf("audio endpoint not found for %s", target)
	}

	// PolicyConfig
	var pc *policyConfig
	hr3 := coCreateInstance(&clsidPolicyConfig, &iidIPolicyConfig, unsafe.Pointer(&pc))
	if hr3 != 0 {
		return fmt.Errorf("CoCreateInstance PolicyConfig failed: 0x%x", hr3)
	}
	defer syscall.SyscallN(pc.lpVtbl.Release, uintptr(unsafe.Pointer(pc)))

	idPtr, _ := syscall.UTF16PtrFromString(foundID)
	for _, role := range []uint32{0, 1, 2} {
		syscall.SyscallN(pc.lpVtbl.SetDefaultEndpoint, uintptr(unsafe.Pointer(pc)), uintptr(unsafe.Pointer(idPtr)), uintptr(role))
	}
	return nil
}

var ole32 = syscall.NewLazyDLL("ole32.dll")
var procCoCreateInstance = ole32.NewProc("CoCreateInstance")

func coCreateInstance(clsid, iid *windows.GUID, ppv unsafe.Pointer) uintptr {
	r1, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(clsid)),
		0,
		CLSCTX_ALL,
		uintptr(unsafe.Pointer(iid)),
		uintptr(ppv),
	)
	return r1
}

func (a *windowsAudio) SetDefaultOutput(addr string) error {
	target := addr
	if names, addrs, err := EnumeratePairedDevices(); err == nil {
		for i := range addrs {
			if addrs[i] == addr {
				target = names[i]
				break
			}
		}
	}
	var lastErr error
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			time.Sleep(500 * time.Millisecond)
		}
		if err := a.trySetDefaultOutput(strings.ToLower(target)); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}
	return lastErr
}

func runCmd(c string) error {
	// simple placeholder
	return nil
}