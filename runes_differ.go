package terminal

// runesDiffer returns the first element where two slices of runes differ, or
// -1 if the slices are the same length and each rune in each slice is the
// same.  If the two are equal up until one ends, the return will be wherever
// the shortest one ended.
func runesDiffer(a, b []rune) int {
	for i, r := range a {
		if len(b) < i+1 {
			return i
		}
		if r != b[i] {
			return i
		}
	}

	if len(b) > len(a) {
		return len(a)
	}

	return -1
}
