package Utils

import "fmt"

func RemoveChunkInSlice[T any](slice []T, start, size int) ([]T, error) {
	end := start + size

	if start < 0 || size < 0 || end > len(slice) {
		return nil, fmt.Errorf("RemoveChunkInSlice invalid parameters: start=%d, size=%d, len=%d", start, size, len(slice))
	}

	return append(slice[:start], slice[end:]...), nil
}

// returns 3 slices, the left section, middle section, and right section of an original slice.
func MidSectionSplitInSlice[T any](slice []T, start, size int) ([]T, []T, []T, error) {
	end := start + size

	if start < 0 {
		return nil, nil, nil, fmt.Errorf("SliceMidSectionSplit invalid parameter [start less than zero] : start=%d, size=%d, len=%d", start, size, len(slice))
	}

	if size < 0 {
		return nil, nil, nil, fmt.Errorf("SliceMidSectionSplit invalid parameter [size less than zero] : start=%d, size=%d, len=%d", start, size, len(slice))
	}

	if end > len(slice) {
		return nil, nil, nil, fmt.Errorf("SliceMidSectionSplit invalid parameter [end is greater than slice length] : start=%d, size=%d, len=%d", start, size, len(slice))
	}

	return slice[:start], slice[start:end], slice[end:], nil
}
