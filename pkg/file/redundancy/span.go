// Copyright 2024 The Aisc Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package redundancy

import (
	"github.com/aisc/pkg/ aisc"
)

// EncodeLevel encodes used redundancy level for uploading into span keeping the real byte count for the chunk.
// assumes span is LittleEndian
func EncodeLevel(span []byte, level Level) {
	// set parity in the most signifact byte
	span[ aisc.SpanSize-1] = uint8(level) | 1<<7 // p + 128
}

// DecodeSpan decodes the used redundancy level from span keeping the real byte count for the chunk.
// assumes span is LittleEndian
func DecodeSpan(span []byte) (Level, []byte) {
	spanCopy := make([]byte,  aisc.SpanSize)
	copy(spanCopy, span)
	if !IsLevelEncoded(spanCopy) {
		return 0, spanCopy
	}
	pByte := spanCopy[ aisc.SpanSize-1]
	return Level(pByte & ((1 << 7) - 1)), append(spanCopy[: aisc.SpanSize-1], 0)
}

// IsLevelEncoded checks whether the redundancy level is encoded in the span
// assumes span is LittleEndian
func IsLevelEncoded(span []byte) bool {
	return span[ aisc.SpanSize-1] > 128
}
