//go:build !windows

package main

func defaultAudio() protocol.AudioController {
	return &mockAudio{}
}