// 快照差量生成与应用

package snapshot

import "errors"

// 生成快照差量
func createDelta(base, current []byte) []byte {
	delta := make([]byte, 0, len(current))
	for i := 0; i < len(current); {
		if base[i] != current[i] {
			delta = append(delta, base[i]^current[i])
			i++
			continue
		}
		start := i
		for i < len(current) && base[i] == current[i] && i-start < 255 {
			i++
		}
		delta = append(delta, 0, uint8(i-start))
	}
	return delta
}

// 应用快照差量
func applyDelta(base, delta []byte) ([]byte, error) {
	var length int
	for i := 0; i < len(delta); i++ {
		if delta[i] == 0 {
			i++
			if i == len(delta) || delta[i] == 0 {
				return nil, errors.New("invalid snapshot delta run")
			}
			length += int(delta[i])
		} else {
			length++
		}
		if length > MaxDataBytes {
			return nil, errors.New("expanded snapshot exceeds the legacy size limit")
		}
	}
	data := make([]byte, length)
	copy(data, base)
	for i, offset := 0, 0; i < len(delta); i++ {
		if delta[i] == 0 {
			i++
			offset += int(delta[i])
			continue
		}
		data[offset] ^= delta[i]
		offset++
	}
	return data, nil
}
