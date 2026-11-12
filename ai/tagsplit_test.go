package ai_test

import (
	"strings"
	"testing"

	"github.com/kataras/iris/v14/ai"
)

// render feeds every chunk through the splitter and returns the text and the
// reasoning it separated out.
func render(s *ai.TagSplitter, chunks ...string) (text, reasoning string) {
	var t, r strings.Builder

	collect := func(parts []ai.Part) {
		for _, p := range parts {
			switch p.Kind() {
			case ai.KindText:
				t.WriteString(p.Delta())
			case ai.KindReasoning:
				r.WriteString(p.Delta())
			}
		}
	}

	for _, chunk := range chunks {
		collect(s.Write(chunk))
	}
	collect(s.Flush())

	return t.String(), r.String()
}

func TestTagSplitWholeTagInOneChunk(t *testing.T) {
	text, reasoning := render(ai.TagSplit("m", "<think>", "</think>"),
		"before <think>pondering</think> after")

	if text != "before  after" {
		t.Errorf("text = %q", text)
	}
	if reasoning != "pondering" {
		t.Errorf("reasoning = %q", reasoning)
	}
}

// TestTagSplitAcrossChunkBoundaries is the case that makes this type necessary.
// A model emits the tag in pieces as readily as in one delta, and a
// strings.Contains on each chunk both misses the tag and leaks its characters
// into the answer.
func TestTagSplitAcrossChunkBoundaries(t *testing.T) {
	text, reasoning := render(ai.TagSplit("m", "<think>", "</think>"),
		"hello <", "th", "ink>deep ", "thought</", "thi", "nk> world")

	if text != "hello  world" {
		t.Errorf("text = %q, want the tag characters kept out of the answer", text)
	}
	if reasoning != "deep thought" {
		t.Errorf("reasoning = %q", reasoning)
	}
}

// TestTagSplitOneCharacterAtATime is the pathological version of the same idea.
func TestTagSplitOneCharacterAtATime(t *testing.T) {
	input := "a<think>b</think>c"

	chunks := make([]string, 0, len(input))
	for _, r := range input {
		chunks = append(chunks, string(r))
	}

	text, reasoning := render(ai.TagSplit("m", "<think>", "</think>"), chunks...)

	if text != "ac" {
		t.Errorf("text = %q, want %q", text, "ac")
	}
	if reasoning != "b" {
		t.Errorf("reasoning = %q, want %q", reasoning, "b")
	}
}

// TestTagSplitStartInside covers models that emit a closing tag having never
// emitted an opening one, which is a real dialect and otherwise baffling.
func TestTagSplitStartInside(t *testing.T) {
	text, reasoning := render(ai.TagSplit("m", "<think>", "</think>").StartInside(),
		"reasoning first</think>then the answer")

	if reasoning != "reasoning first" {
		t.Errorf("reasoning = %q", reasoning)
	}
	if text != "then the answer" {
		t.Errorf("text = %q", text)
	}
}

// TestTagSplitUnterminatedTagIsNotATag: characters held back waiting for a tag
// that never completes are part of the answer. Dropping them would silently
// truncate it.
func TestTagSplitUnterminatedTagIsNotATag(t *testing.T) {
	text, reasoning := render(ai.TagSplit("m", "<think>", "</think>"),
		"answer ending in <thin")

	if text != "answer ending in <thin" {
		t.Errorf("text = %q, want the incomplete tag returned as text", text)
	}
	if reasoning != "" {
		t.Errorf("reasoning = %q, want none", reasoning)
	}
}

func TestTagSplitUnclosedReasoningIsFlushed(t *testing.T) {
	text, reasoning := render(ai.TagSplit("m", "<think>", "</think>"),
		"a<think>never closed")

	if text != "a" {
		t.Errorf("text = %q", text)
	}
	if reasoning != "never closed" {
		t.Errorf("reasoning = %q, want the unterminated reasoning flushed", reasoning)
	}
}

func TestTagSplitMultipleRuns(t *testing.T) {
	text, reasoning := render(ai.TagSplit("m", "<think>", "</think>"),
		"one<think>A</think>two<think>B</think>three")

	if text != "onetwothree" {
		t.Errorf("text = %q", text)
	}
	if reasoning != "AB" {
		t.Errorf("reasoning = %q", reasoning)
	}
}

func TestTagSplitInsideReporting(t *testing.T) {
	s := ai.TagSplit("m", "<think>", "</think>")

	s.Write("plain")
	if s.Inside() {
		t.Error("Inside reported true outside the tag")
	}

	s.Write("<think>x")
	if !s.Inside() {
		t.Error("Inside reported false inside the tag")
	}

	s.Write("</think>y")
	if s.Inside() {
		t.Error("Inside reported true after the tag closed")
	}
}

func TestTagSplitCarriesTheID(t *testing.T) {
	parts := ai.TagSplit("msg-7", "<think>", "</think>").Write("hello")

	if len(parts) != 1 {
		t.Fatalf("got %d parts, want 1", len(parts))
	}
	if parts[0].ID() != "msg-7" {
		t.Errorf("id = %q, want msg-7", parts[0].ID())
	}
}

func TestTagSplitEmptyWrite(t *testing.T) {
	if parts := ai.TagSplit("m", "<think>", "</think>").Write(""); parts != nil {
		t.Fatalf("Write(\"\") = %v, want nothing", parts)
	}
}

// TestTagSplitRejectsEmptyMarkers: an empty marker is found at every position,
// so the splitter would loop forever or call the whole answer reasoning. It is a
// programming error and is reported as one.
func TestTagSplitRejectsEmptyMarkers(t *testing.T) {
	for name, markers := range map[string][2]string{
		"empty open":  {"", "</think>"},
		"empty close": {"<think>", ""},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("TagSplit accepted an empty marker")
				}
			}()
			ai.TagSplit("m", markers[0], markers[1])
		})
	}
}
