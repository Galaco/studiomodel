package internal

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// CheckRange returns an error unless [offset, offset+size) lies within buf.
func CheckRange(buf []byte, offset int64, size int64, name string) error {
	if offset < 0 {
		return fmt.Errorf("%s offset is negative: %d", name, offset)
	}
	if size < 0 {
		return fmt.Errorf("%s size is negative: %d", name, size)
	}
	if offset+size > int64(len(buf)) {
		return fmt.Errorf("%s data exceeds buffer: offset=%d size=%d required=%d bufferSize=%d", name, offset, size, offset+size, len(buf))
	}
	return nil
}

// ReadStruct reads a single little-endian T at offset.
func ReadStruct[T any](buf []byte, offset int64, name string) (T, error) {
	var out T
	size := int64(binary.Size(out))
	if err := CheckRange(buf, offset, size, name); err != nil {
		return out, err
	}
	if err := binary.Read(bytes.NewReader(buf[offset:offset+size]), binary.LittleEndian, &out); err != nil {
		return out, fmt.Errorf("failed to read %s at offset %d: %w", name, offset, err)
	}
	return out, nil
}

// ReadSlice reads count contiguous little-endian Ts starting at offset.
// Bounds are checked before anything is allocated, so a corrupt count cannot cause a huge allocation.
func ReadSlice[T any](buf []byte, offset int64, count int64, name string) ([]T, error) {
	var zero T
	return ReadStrided[T](buf, offset, count, int64(binary.Size(zero)), name)
}

// ReadStrided reads count little-endian Ts starting at offset, with each element stride bytes after the previous.
// Any bytes between the end of one T and the start of the next are skipped.
func ReadStrided[T any](buf []byte, offset int64, count int64, stride int64, name string) ([]T, error) {
	var zero T
	size := int64(binary.Size(zero))
	if count < 0 {
		return nil, fmt.Errorf("%s count is negative: %d", name, count)
	}
	if count == 0 {
		return []T{}, nil
	}
	if stride < size {
		return nil, fmt.Errorf("%s stride %d is smaller than element size %d", name, stride, size)
	}
	total := (count-1)*stride + size
	if err := CheckRange(buf, offset, total, name); err != nil {
		return nil, err
	}

	out := make([]T, count)
	if stride == size {
		if err := binary.Read(bytes.NewReader(buf[offset:offset+total]), binary.LittleEndian, out); err != nil {
			return nil, fmt.Errorf("failed to read %s at offset %d: %w", name, offset, err)
		}
		return out, nil
	}
	for i := range out {
		start := offset + int64(i)*stride
		if err := binary.Read(bytes.NewReader(buf[start:start+size]), binary.LittleEndian, &out[i]); err != nil {
			return nil, fmt.Errorf("failed to read %s %d at offset %d: %w", name, i, start, err)
		}
	}
	return out, nil
}

// CString reads a null-terminated string at offset, stopping after maxLen bytes or at the end of buf.
func CString(buf []byte, offset int64, maxLen int, name string) (string, error) {
	if offset < 0 || offset >= int64(len(buf)) {
		return "", fmt.Errorf("%s string offset %d out of bounds (buffer size %d)", name, offset, len(buf))
	}
	end := offset
	limit := min(offset+int64(maxLen), int64(len(buf)))
	for end < limit && buf[end] != 0 {
		end++
	}
	return string(buf[offset:end]), nil
}
