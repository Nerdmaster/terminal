// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package terminal

import (
	"io"
	"syscall"
	"unsafe"
)

const (
	enableLineInput       = 2
	enableEchoInput       = 4
	enableProcessedInput  = 1
	enableWindowInput     = 8
	enableMouseInput      = 16
	enableInsertMode      = 32
	enableQuickEditMode   = 64
	enableExtendedFlags   = 128
	enableAutoPosition    = 256
	enableProcessedOutput = 1
	enableWrapAtEolOutput = 2
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

var (
	procGetConsoleMode             = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
)

type (
	short int16
	word  uint16

	coord struct {
		x short
		y short
	}
	smallRect struct {
		left   short
		top    short
		right  short
		bottom short
	}
	consoleScreenBufferInfo struct {
		size              coord
		cursorPosition    coord
		attributes        word
		window            smallRect
		maximumWindowSize coord
	}
)

type State struct {
	mode uint32
}

// VisualLength returns the number of visible glyphs in a string.  This can be
// useful for getting the length of a string which has ANSI color sequences,
// but it doesn't count "wide" glyphs differently than others, and it won't
// handle ANSI cursor commands; e.g., it ignores "\x1b[D" rather than knowing
// that the cursor position moved to the left.
func VisualLength(s string) int {
	runes := []rune(s)
	inEscapeSeq := false
	length := 0

	for _, r := range runes {
		switch {
		case inEscapeSeq:
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscapeSeq = false
			}
		case r == '\x1b':
			inEscapeSeq = true
		default:
			length++
		}
	}

	return length
}

// IsTerminal returns true if the given file descriptor is a terminal.
func IsTerminal(fd int) bool {
	var st uint32
	r, _, e := syscall.Syscall(procGetConsoleMode.Addr(), 2, uintptr(fd), uintptr(unsafe.Pointer(&st)), 0)
	return r != 0 && e == 0
}

// MakeRaw put the terminal connected to the given file descriptor into raw
// mode and returns the previous state of the terminal so that it can be
// restored.
func MakeRaw(fd int) (*State, error) {
	var st uint32
	_, _, e := syscall.Syscall(procGetConsoleMode.Addr(), 2, uintptr(fd), uintptr(unsafe.Pointer(&st)), 0)
	if e != 0 {
		return nil, error(e)
	}
	raw := st &^ (enableEchoInput | enableProcessedInput | enableLineInput | enableProcessedOutput)
	_, _, e = syscall.Syscall(procSetConsoleMode.Addr(), 2, uintptr(fd), uintptr(raw), 0)
	if e != 0 {
		return nil, error(e)
	}
	return &State{st}, nil
}

// GetState returns the current state of a terminal which may be useful to
// restore the terminal after a signal.
func GetState(fd int) (*State, error) {
	var st uint32
	_, _, e := syscall.Syscall(procGetConsoleMode.Addr(), 2, uintptr(fd), uintptr(unsafe.Pointer(&st)), 0)
	if e != 0 {
		return nil, error(e)
	}
	return &State{st}, nil
}

// Restore restores the terminal connected to the given file descriptor to a
// previous state.
func Restore(fd int, state *State) error {
	_, _, err := syscall.Syscall(procSetConsoleMode.Addr(), 2, uintptr(fd), uintptr(state.mode), 0)
	return err
}

// GetSize returns the dimensions of the given terminal.
func GetSize(fd int) (width, height int, err error) {
	var info consoleScreenBufferInfo
	_, _, e := syscall.Syscall(procGetConsoleScreenBufferInfo.Addr(), 2, uintptr(fd), uintptr(unsafe.Pointer(&info)), 0)
	if e != 0 {
		return 0, 0, error(e)
	}
	return int(info.size.x), int(info.size.y), nil
}

// ReadPassword reads a line of input from a terminal without local echo.  This
// is commonly used for inputting passwords and other sensitive data. The slice
// returned does not include the \n.
func ReadPassword(fd int) ([]byte, error) {
	var st uint32
	_, _, e := syscall.Syscall(procGetConsoleMode.Addr(), 2, uintptr(fd), uintptr(unsafe.Pointer(&st)), 0)
	if e != 0 {
		return nil, error(e)
	}
	old := st

	st &^= (enableEchoInput)
	st |= (enableProcessedInput | enableLineInput | enableProcessedOutput)
	_, _, e = syscall.Syscall(procSetConsoleMode.Addr(), 2, uintptr(fd), uintptr(st), 0)
	if e != 0 {
		return nil, error(e)
	}

	defer func() {
		syscall.Syscall(procSetConsoleMode.Addr(), 2, uintptr(fd), uintptr(old), 0)
	}()

	var buf [16]byte
	var ret []byte
	for {
		n, err := syscall.Read(syscall.Handle(fd), buf[:])
		if err != nil {
			return nil, err
		}
		if n == 0 {
			if len(ret) == 0 {
				return nil, io.EOF
			}
			break
		}
		if buf[n-1] == '\n' {
			n--
		}
		if n > 0 && buf[n-1] == '\r' {
			n--
		}
		ret = append(ret, buf[:n]...)
		if n < len(buf) {
			break
		}
	}

	return ret, nil
}
