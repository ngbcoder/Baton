//go:build !windows

package main

func enableStartAtLogin() error  { return nil }
func disableStartAtLogin() error { return nil }