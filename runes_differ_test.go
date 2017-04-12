package terminal

import (
	"testing"
)

func TestRunesDiffer(t *testing.T) {
	var a = []rune("test 1")
	var b = []rune("test 2")

	if runesDiffer(a, b) != 5 {
		t.Error("Expected difference to occur at index 5")
	}

	b[5] = '1'
	if runesDiffer(a, b) != -1 {
		t.Error("Expected no differences")
	}

	b = []rune("test 123")
	if runesDiffer(a, b) != 6 {
		t.Error("Expected difference at index 6 (one character beyond a)")
	}

	b[3] = 'X'
	if runesDiffer(a, b) != 3 {
		t.Error("Expected difference at index 3 (mismatched length doesn't matter if they differ early)")
	}
}
