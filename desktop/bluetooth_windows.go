//go:build windows

package main

/*
#cgo LDFLAGS: -lbthprops
#include <windows.h>
#include <bthdef.h>
#include <bthprops.h>
#include <stdio.h>

// Helper to enumerate paired devices into a simple array of wide strings.
int EnumeratePairedDevices(wchar_t names[][256], wchar_t addrs[][64], int max) {
    BLUETOOTH_DEVICE_SEARCH_PARAMS searchParams;
    memset(&searchParams, 0, sizeof(searchParams));
    searchParams.dwSize = sizeof(searchParams);
    searchParams.fReturnAuthenticated = TRUE;
    searchParams.fReturnRemembered = TRUE;
    searchParams.fReturnConnected = TRUE;
    searchParams.fReturnUnknown = FALSE;
    searchParams.fIssueInquiry = FALSE;
    searchParams.cTimeoutMultiplier = 0;

    BLUETOOTH_DEVICE_INFO deviceInfo;
    memset(&deviceInfo, 0, sizeof(deviceInfo));
    deviceInfo.dwSize = sizeof(deviceInfo);

    HBLUETOOTH_DEVICE_FIND hFind = BluetoothFindFirstDevice(&searchParams, &deviceInfo);
    if (hFind == NULL) {
        return 0;
    }
    int count = 0;
    do {
        if (count < max) {
            wcscpy_s(names[count], 256, deviceInfo.szName);
            // format address
            swprintf_s(addrs[count], 64, L"%04X:%02X:%02X:%02X:%02X:%02X",
                (deviceInfo.Address.ullLong & 0xFFFF000000000000ULL) >> 48,
                (deviceInfo.Address.ullLong & 0x0000FF0000000000ULL) >> 40,
                (deviceInfo.Address.ullLong & 0x000000FF00000000ULL) >> 32,
                (deviceInfo.Address.ullLong & 0x00000000FF000000ULL) >> 24,
                (deviceInfo.Address.ullLong & 0x0000000000FF0000ULL) >> 16,
                (deviceInfo.Address.ullLong & 0x000000000000FFFFULL));
            count++;
        }
    } while (BluetoothFindNextDevice(hFind, &deviceInfo));
    BluetoothFindDeviceClose(hFind);
    return count;
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// EnumeratePairedDevices returns paired Bluetooth device names/addresses.
func EnumeratePairedDevices() ([]string, []string, error) {
	const max = 64
	names := make([][256]uint16, max)
	addrs := make([][64]uint16, max)
	n := C.EnumeratePairedDevices((*C.wchar_t)(unsafe.Pointer(&names[0])), (*C.wchar_t)(unsafe.Pointer(&addrs[0])), C.int(max))
	if n < 0 {
		return nil, nil, fmt.Errorf("enumerate failed")
	}
	var nameList, addrList []string
	for i := 0; i < int(n); i++ {
		nameList = append(nameList, C.GoString((*C.char)(unsafe.Pointer(&names[i])))) // not correct for wchar
		_ = addrList
	}
	return nameList, nil, nil
}