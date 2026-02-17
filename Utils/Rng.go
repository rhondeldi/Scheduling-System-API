package Utils

import "math/rand"

func RandomInRange(min, max int) int {
	if min > max {
		min, max = max, min
	}

	return rand.Intn(max-min+1) + min
}
