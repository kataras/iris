package ai

import "strings"

// TagSplitter separates text wrapped in an inline tag from the rest, across
// chunk boundaries.
//
// Several models emit their reasoning inline, fenced by a tag such as <think>,
// rather than in a separate field. The tag arrives split across deltas as often
// as not - "<th" in one chunk and "ink>" in the next - so a naive
// strings.Contains on each chunk both misses the tag and leaks its characters
// into the answer.
//
// It holds a small buffer, not a stream: feed it deltas, take the parts it
// returns. Use it through the pointer TagSplit returns; it is not safe for
// concurrent use.
type TagSplitter struct {
	id          string
	open, close string
	inside      bool
	pending     strings.Builder
}

// TagSplit returns a splitter for text fenced by open and close, emitting the
// fenced text as reasoning and everything else as answer text.
//
//	splitter := ai.TagSplit(msgID, "<think>", "</think>")
//	for _, p := range splitter.Write(delta) {
//		res.Send(p)
//	}
//	// ... at the end of the stream:
//	for _, p := range splitter.Flush() {
//		res.Send(p)
//	}
//
// It panics on an empty marker: an empty string is found at every position, so
// the splitter would loop forever or call the whole answer reasoning, and no
// model emits an invisible tag.
func TagSplit(id, open, close string) *TagSplitter {
	if open == "" || close == "" {
		panic("iris: ai: TagSplit: the open and close markers must not be empty")
	}

	return &TagSplitter{id: id, open: open, close: close}
}

// StartInside tells the splitter that the stream begins inside the tag.
//
// It exists for models that emit a closing tag without ever having emitted an
// opening one, which is a real and otherwise baffling dialect: everything before
// the first </think> is reasoning.
func (s *TagSplitter) StartInside() *TagSplitter {
	s.inside = true
	return s
}

// Inside reports whether the splitter is currently inside the tag.
func (s *TagSplitter) Inside() bool { return s.inside }

// Write feeds one delta and returns the parts it completed.
//
// It holds back only what could still turn out to be the start of a tag, so a
// tag split across any number of chunks is recognised, and at most len(tag)-1
// characters are ever delayed.
func (s *TagSplitter) Write(delta string) []Part {
	if delta == "" {
		return nil
	}

	s.pending.WriteString(delta)
	buffered := s.pending.String()
	s.pending.Reset()

	var parts []Part

	for {
		marker := s.close
		if !s.inside {
			marker = s.open
		}

		idx := strings.Index(buffered, marker)
		if idx < 0 {
			break
		}

		if text := buffered[:idx]; text != "" {
			parts = append(parts, s.emit(text))
		}

		buffered = buffered[idx+len(marker):]
		s.inside = !s.inside
	}

	// Whatever is left may still be the beginning of the marker we are looking
	// for. Hold back exactly that much and release the rest.
	marker := s.close
	if !s.inside {
		marker = s.open
	}

	hold := partialSuffix(buffered, marker)
	if emit := buffered[:len(buffered)-hold]; emit != "" {
		parts = append(parts, s.emit(emit))
	}
	s.pending.WriteString(buffered[len(buffered)-hold:])

	return parts
}

// Flush releases whatever is still held back, at the end of a stream.
//
// A tag that never completed is not a tag: its characters are part of the
// answer, and dropping them would silently truncate it.
func (s *TagSplitter) Flush() []Part {
	rest := s.pending.String()
	s.pending.Reset()

	if rest == "" {
		return nil
	}

	return []Part{s.emit(rest)}
}

func (s *TagSplitter) emit(text string) Part {
	if s.inside {
		return Reasoning(s.id, text)
	}

	return Text(s.id, text)
}

// partialSuffix returns the length of the longest suffix of s that is a proper
// prefix of marker, which is exactly how much must be held back.
func partialSuffix(s, marker string) int {
	max := len(marker) - 1
	if max > len(s) {
		max = len(s)
	}

	for n := max; n > 0; n-- {
		if strings.HasPrefix(marker, s[len(s)-n:]) {
			return n
		}
	}

	return 0
}
