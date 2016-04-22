// Copyright 2011 The Snappy-Go Authors. All rights reserved.
// Modified for deflate by Klaus Post (c) 2015.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flate

// We limit how far copy back-references can go, the same as the C++ code.
const maxOffset = 1 << 15

// emitLiteral writes a literal chunk and returns the number of bytes written.
func emitLiteral(dst *tokens, lit []byte) {
	ol := dst.n
	for i, v := range lit {
		dst.tokens[i+ol] = token(v)
	}
	dst.n += len(lit)
}

// emitCopy writes a copy chunk and returns the number of bytes written.
func emitCopy(dst *tokens, offset, length int) {
	dst.tokens[dst.n] = matchToken(uint32(length-3), uint32(offset-minOffsetSize))
	dst.n++
}

type snappyEnc interface {
	Encode(dst *tokens, src []byte)
	Reset()
}

func newSnappy(level int) snappyEnc {
	if useSSE42 {
		e := &snappySSE4{snappyGen: snappyGen{cur: 1}}
		switch level {
		case 3:
			e.enc = e.encodeL3
			return e
		}
	}
	e := &snappyGen{cur: 1}
	switch level {
	case 1:
		e.enc = e.encodeL1
	case 2:
		e.enc = e.encodeL2
	case 3:
		e.enc = e.encodeL3
	default:
		panic("invalid level specified")
	}
	return e
}

const tableBits = 14             // Bits used in the table
const tableSize = 1 << tableBits // Size of the table

// snappyGen maintains the table for matches,
// and the previous byte block for level 2.
// This is the generic implementation.
type snappyGen struct {
	table [tableSize]int64
	block [maxStoreBlockSize]byte
	prev  []byte
	cur   int
	enc   func(dst *tokens, src []byte)
}

func (e *snappyGen) Encode(dst *tokens, src []byte) {
	e.enc(dst, src)
}

// EncodeL1 uses Snappy-like compression, but stores as Huffman
// blocks.
func (e *snappyGen) encodeL1(dst *tokens, src []byte) {
	// Return early if src is short.
	if len(src) <= 4 {
		if len(src) != 0 {
			emitLiteral(dst, src)
		}
		e.cur += 4
		return
	}

	// Ensure that e.cur doesn't wrap, mainly an issue on 32 bits.
	if e.cur > 1<<30 {
		e.cur = 1
	}

	// Iterate over the source bytes.
	var (
		s   int // The iterator position.
		t   int // The last position with the same hash as s.
		lit int // The start position of any pending literal bytes.
	)

	for s+3 < len(src) {
		// Update the hash table.
		b0, b1, b2, b3 := src[s], src[s+1], src[s+2], src[s+3]
		h := uint32(b0) | uint32(b1)<<8 | uint32(b2)<<16 | uint32(b3)<<24
		p := &e.table[(h*0x1e35a7bd)>>(32-tableBits)]
		// We need to to store values in [-1, inf) in table.
		// To save some initialization time, we make sure that
		// e.cur is never zero.
		t, *p = int(*p)-e.cur, int64(s+e.cur)

		offset := uint(s - t - 1)

		// If t is invalid or src[s:s+4] differs from src[t:t+4], accumulate a literal byte.
		if t < 0 || offset >= (maxOffset-1) || b0 != src[t] || b1 != src[t+1] || b2 != src[t+2] || b3 != src[t+3] {
			// Skip 1 byte for 16 consecutive missed.
			s += 1 + ((s - lit) >> 4)
			continue
		}
		// Otherwise, we have a match. First, emit any pending literal bytes.
		if lit != s {
			emitLiteral(dst, src[lit:s])
		}
		// Extend the match to be as long as possible.
		s0 := s
		s1 := s + maxMatchLength
		if s1 > len(src) {
			s1 = len(src)
		}
		s, t = s+4, t+4
		for s < s1 && src[s] == src[t] {
			s++
			t++
		}
		// Emit the copied bytes.
		// inlined: emitCopy(dst, s-t, s-s0)

		dst.tokens[dst.n] = matchToken(uint32(s-s0-3), uint32(s-t-minOffsetSize))
		dst.n++
		lit = s
	}

	// Emit any final pending literal bytes and return.
	if lit != len(src) {
		emitLiteral(dst, src[lit:])
	}
	e.cur += len(src)
}

// EncodeL2 uses a similar algorithm to level 1, but is capable
// of matching across blocks giving better compression at a small slowdown.
func (e *snappyGen) encodeL2(dst *tokens, src []byte) {
	// Return early if src is short.
	if len(src) <= 4 {
		if len(src) != 0 {
			emitLiteral(dst, src)
		}
		e.prev = nil
		e.cur += len(src)
		return
	}

	// Ensure that e.cur doesn't wrap, mainly an issue on 32 bits.
	if e.cur > 1<<30 {
		e.cur = 1
	}

	// Iterate over the source bytes.
	var (
		s   int // The iterator position.
		t   int // The last position with the same hash as s.
		lit int // The start position of any pending literal bytes.
	)

	for s+3 < len(src) {
		// Update the hash table.
		b0, b1, b2, b3 := src[s], src[s+1], src[s+2], src[s+3]
		h := uint32(b0) | uint32(b1)<<8 | uint32(b2)<<16 | uint32(b3)<<24
		p := &e.table[(h*0x1e35a7bd)>>(32-tableBits)]
		// We need to to store values in [-1, inf) in table.
		// To save some initialization time, we make sure that
		// e.cur is never zero.
		t, *p = int(*p)-e.cur, int64(s+e.cur)

		// If t is positive, the match starts in the current block
		if t >= 0 {

			offset := uint(s - t - 1)
			// Check that the offset is valid and that we match at least 4 bytes
			if offset >= (maxOffset-1) || b0 != src[t] || b1 != src[t+1] || b2 != src[t+2] || b3 != src[t+3] {
				// Skip 1 byte for 32 consecutive missed.
				s += 1 + ((s - lit) >> 5)
				continue
			}
			// Otherwise, we have a match. First, emit any pending literal bytes.
			if lit != s {
				emitLiteral(dst, src[lit:s])
			}
			// Extend the match to be as long as possible.
			s0 := s
			s1 := s + maxMatchLength
			if s1 > len(src) {
				s1 = len(src)
			}
			s, t = s+4, t+4
			for s < s1 && src[s] == src[t] {
				s++
				t++
			}
			// Emit the copied bytes.
			// inlined: emitCopy(dst, s-t, s-s0)
			dst.tokens[dst.n] = matchToken(uint32(s-s0-3), uint32(s-t-minOffsetSize))
			dst.n++
			lit = s
			continue
		}
		// We found a match in the previous block.
		tp := len(e.prev) + t
		if tp < 0 || t > -5 || s-t >= maxOffset || b0 != e.prev[tp] || b1 != e.prev[tp+1] || b2 != e.prev[tp+2] || b3 != e.prev[tp+3] {
			// Skip 1 byte for 32 consecutive missed.
			s += 1 + ((s - lit) >> 5)
			continue
		}
		// Otherwise, we have a match. First, emit any pending literal bytes.
		if lit != s {
			emitLiteral(dst, src[lit:s])
		}
		// Extend the match to be as long as possible.
		s0 := s
		s1 := s + maxMatchLength
		if s1 > len(src) {
			s1 = len(src)
		}
		s, tp = s+4, tp+4
		for s < s1 && src[s] == e.prev[tp] {
			s++
			tp++
			if tp == len(e.prev) {
				t = 0
				// continue in current buffer
				for s < s1 && src[s] == src[t] {
					s++
					t++
				}
				goto l
			}
		}
	l:
		// Emit the copied bytes.
		if t < 0 {
			t = tp - len(e.prev)
		}
		dst.tokens[dst.n] = matchToken(uint32(s-s0-3), uint32(s-t-minOffsetSize))
		dst.n++
		lit = s

	}

	// Emit any final pending literal bytes and return.
	if lit != len(src) {
		emitLiteral(dst, src[lit:])
	}
	e.cur += len(src)
	// Store this block, if it was full length.
	if len(src) == maxStoreBlockSize {
		copy(e.block[:], src)
		e.prev = e.block[:len(src)]
	} else {
		e.prev = nil
	}
}

// EncodeL3 uses a similar algorithm to level 2, but is capable
// will keep two matches per hash.
// Both hashes are checked if the first isn't ok, and the longest is selected.
func (e *snappyGen) encodeL3(dst *tokens, src []byte) {
	// Return early if src is short.
	if len(src) <= 4 {
		if len(src) != 0 {
			emitLiteral(dst, src)
		}
		e.prev = nil
		e.cur += len(src)
		return
	}

	// Ensure that e.cur doesn't wrap, mainly an issue on 32 bits.
	if e.cur > 1<<30 {
		e.cur = 1
	}

	// Iterate over the source bytes.
	var (
		s   int // The iterator position.
		lit int // The start position of any pending literal bytes.
	)

	for s+3 < len(src) {
		// Update the hash table.
		h := uint32(src[s]) | uint32(src[s+1])<<8 | uint32(src[s+2])<<16 | uint32(src[s+3])<<24
		p := &e.table[(h*0x1e35a7bd)>>(32-tableBits)]
		tmp := *p
		p1 := int(tmp & 0xffffffff) // Closest match position
		p2 := int(tmp >> 32)        // Furthest match position

		// We need to to store values in [-1, inf) in table.
		// To save some initialization time, we make sure that
		// e.cur is never zero.
		t1 := p1 - e.cur

		var l2 int
		var t2 int
		l1 := e.matchlen(s, t1, src)
		// If fist match was ok, don't do the second.
		if l1 < 16 {
			t2 = p2 - e.cur
			l2 = e.matchlen(s, t2, src)

			// If both are short, continue
			if l1 < 4 && l2 < 4 {
				// Update hash table
				*p = int64(s+e.cur) | (int64(p1) << 32)
				// Skip 1 byte for 32 consecutive missed.
				s += 1 + ((s - lit) >> 5)
				continue
			}
		}

		// Otherwise, we have a match. First, emit any pending literal bytes.
		if lit != s {
			emitLiteral(dst, src[lit:s])
		}
		// Update hash table
		*p = int64(s+e.cur) | (int64(p1) << 32)

		// Store the longest match l1 will be closest, so we prefer that if equal length
		if l1 >= l2 {
			dst.tokens[dst.n] = matchToken(uint32(l1-3), uint32(s-t1-minOffsetSize))
			s += l1
		} else {
			dst.tokens[dst.n] = matchToken(uint32(l2-3), uint32(s-t2-minOffsetSize))
			s += l2
		}
		dst.n++
		lit = s
	}

	// Emit any final pending literal bytes and return.
	if lit != len(src) {
		emitLiteral(dst, src[lit:])
	}
	e.cur += len(src)
	// Store this block, if it was full length.
	if len(src) == maxStoreBlockSize {
		copy(e.block[:], src)
		e.prev = e.block[:len(src)]
	} else {
		e.prev = nil
	}
}

func (e *snappyGen) matchlen(s, t int, src []byte) int {
	// If t is invalid or src[s:s+4] differs from src[t:t+4], accumulate a literal byte.
	offset := uint(s - t - 1)

	// If we are inside the current block
	if t >= 0 {
		if offset >= (maxOffset-1) ||
			src[s] != src[t] || src[s+1] != src[t+1] ||
			src[s+2] != src[t+2] || src[s+3] != src[t+3] {
			return 0
		}
		// Extend the match to be as long as possible.
		s0 := s
		s1 := s + maxMatchLength
		if s1 > len(src) {
			s1 = len(src)
		}
		s, t = s+4, t+4
		for s < s1 && src[s] == src[t] {
			s++
			t++
		}
		return s - s0
	}

	// We found a match in the previous block.
	tp := len(e.prev) + t
	if tp < 0 || offset >= (maxOffset-1) || t > -5 ||
		src[s] != e.prev[tp] || src[s+1] != e.prev[tp+1] ||
		src[s+2] != e.prev[tp+2] || src[s+3] != e.prev[tp+3] {
		return 0
	}

	// Extend the match to be as long as possible.
	s0 := s
	s1 := s + maxMatchLength
	if s1 > len(src) {
		s1 = len(src)
	}
	s, tp = s+4, tp+4
	for s < s1 && src[s] == e.prev[tp] {
		s++
		tp++
		if tp == len(e.prev) {
			t = 0
			// continue in current buffer
			for s < s1 && src[s] == src[t] {
				s++
				t++
			}
			return s - s0
		}
	}
	return s - s0
}

// Reset the encoding table.
func (e *snappyGen) Reset() {
	e.prev = nil
}

// snappySSE4 extends snappyGen.
// This implementation can use SSE 4.2 for length matching.
type snappySSE4 struct {
	snappyGen
}

// EncodeL3 uses a similar algorithm to level 2,
// but will keep two matches per hash.
// Both hashes are checked if the first isn't ok, and the longest is selected.
func (e *snappySSE4) encodeL3(dst *tokens, src []byte) {
	// Return early if src is short.
	if len(src) <= 4 {
		if len(src) != 0 {
			emitLiteral(dst, src)
		}
		e.prev = nil
		e.cur += len(src)
		return
	}

	// Ensure that e.cur doesn't wrap, mainly an issue on 32 bits.
	if e.cur > 1<<30 {
		e.cur = 1
	}

	// Iterate over the source bytes.
	var (
		s   int // The iterator position.
		lit int // The start position of any pending literal bytes.
	)

	for s+3 < len(src) {
		// Load potential matches from hash table.
		h := uint32(src[s]) | uint32(src[s+1])<<8 | uint32(src[s+2])<<16 | uint32(src[s+3])<<24
		p := &e.table[(h*0x1e35a7bd)>>(32-tableBits)]
		tmp := *p
		p1 := int(tmp & 0xffffffff) // Closest match position
		p2 := int(tmp >> 32)        // Furthest match position

		// We need to to store values in [-1, inf) in table.
		// To save some initialization time, we make sure that
		// e.cur is never zero.
		t1 := int(p1) - e.cur

		var l2 int
		var t2 int
		l1 := e.matchlen(s, t1, src)
		// If fist match was ok, don't do the second.
		if l1 < 16 {
			t2 = int(p2) - e.cur
			l2 = e.matchlen(s, t2, src)

			// If both are short, continue
			if l1 < 4 && l2 < 4 {
				// Update hash table
				*p = int64(s+e.cur) | (int64(p1) << 32)
				// Skip 1 byte for 32 consecutive missed.
				s += 1 + ((s - lit) >> 5)
				continue
			}
		}

		// Otherwise, we have a match. First, emit any pending literal bytes.
		if lit != s {
			emitLiteral(dst, src[lit:s])
		}
		// Update hash table
		*p = int64(s+e.cur) | (int64(p1) << 32)

		// Store the longest match l1 will be closest, so we prefer that if equal length
		if l1 >= l2 {
			dst.tokens[dst.n] = matchToken(uint32(l1-3), uint32(s-t1-minOffsetSize))
			s += l1
		} else {
			dst.tokens[dst.n] = matchToken(uint32(l2-3), uint32(s-t2-minOffsetSize))
			s += l2
		}
		dst.n++
		lit = s
	}

	// Emit any final pending literal bytes and return.
	if lit != len(src) {
		emitLiteral(dst, src[lit:])
	}
	e.cur += len(src)
	// Store this block, if it was full length.
	if len(src) == maxStoreBlockSize {
		copy(e.block[:], src)
		e.prev = e.block[:len(src)]
	} else {
		e.prev = nil
	}
}

func (e *snappySSE4) matchlen(s, t int, src []byte) int {
	// If t is invalid or src[s:s+4] differs from src[t:t+4], accumulate a literal byte.
	offset := uint(s - t - 1)

	// If we are inside the current block
	if t >= 0 {
		if offset >= (maxOffset - 1) {
			return 0
		}
		length := len(src) - s
		if length > maxMatchLength {
			length = maxMatchLength
		}
		// Extend the match to be as long as possible.
		return matchLenSSE4(src[t:], src[s:], length)
	}

	// We found a match in the previous block.
	tp := len(e.prev) + t
	if tp < 0 || offset >= (maxOffset-1) || t > -5 ||
		src[s] != e.prev[tp] || src[s+1] != e.prev[tp+1] ||
		src[s+2] != e.prev[tp+2] || src[s+3] != e.prev[tp+3] {
		return 0
	}

	// Extend the match to be as long as possible.
	s0 := s
	s1 := s + maxMatchLength
	if s1 > len(src) {
		s1 = len(src)
	}
	s, tp = s+4, tp+4
	for s < s1 && src[s] == e.prev[tp] {
		s++
		tp++
		if tp == len(e.prev) {
			t = 0
			// continue in current buffer
			for s < s1 && src[s] == src[t] {
				s++
				t++
			}
			return s - s0
		}
	}
	return s - s0
}
