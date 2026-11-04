//go:build !windows

package main

func defaultBluetooth() protocol.BluetoothController {
	return newMockBluetooth()
}