package entity

/*
 * Compare 2 string arrays to see if they contains the same elements
 * Returns
 * - true if both are the same
 * - left only
 * - rigght only
 */
func StringArrayEquivalent(a, b []string) (bool, []string, []string) {
	leftOnly := []string{}
	rightOnly := []string{}
	lefts := make(map[string]bool)
	for _, m := range a {
		lefts[m] = true
	}

	rights := make(map[string]bool)
	for _, m := range b {
		rights[m] = true
	}

	result := true

	if len(lefts) != len(rights) {
		result = false
	}

	for r, _ := range rights {
		if _, ok := lefts[r]; !ok {
			leftOnly = append(leftOnly, r)
			result = false
		}
	}
	for l, _ := range lefts {
		if _, ok := rights[l]; !ok {
			rightOnly = append(rightOnly, l)
			result = false
		}
	}
	return result, leftOnly, rightOnly
}
