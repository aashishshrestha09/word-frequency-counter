package counter

import (
	"sync"
	"testing"
)

func TestPartitionLines(t *testing.T) {
	tests := []struct {
		name        string
		lines       []string
		numSegments int
		wantLen     int
	}{
		{
			name:        "Simple partition",
			lines:       []string{"a", "b", "c", "d"},
			numSegments: 2,
			wantLen:     2,
		},
		{
			name:        "Uneven partition",
			lines:       []string{"a", "b", "c", "d", "e"},
			numSegments: 2,
			wantLen:     2,
		},
		{
			name:        "More segments than lines",
			lines:       []string{"a", "b"},
			numSegments: 5,
			wantLen:     2,
		},
		{
			name:        "Single segment",
			lines:       []string{"a", "b", "c"},
			numSegments: 1,
			wantLen:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := PartitionLines(tt.lines, tt.numSegments)
			if len(segments) != tt.wantLen {
				t.Errorf("PartitionLines() got %d segments, want %d", len(segments), tt.wantLen)
			}

			// Verify all lines are accounted for
			totalLines := 0
			for _, seg := range segments {
				totalLines += len(seg)
			}
			if totalLines != len(tt.lines) {
				t.Errorf("Total lines in segments = %d, want %d", totalLines, len(tt.lines))
			}
		})
	}
}

func TestProcessSegment(t *testing.T) {
	counter := NewCounter()
	lines := []string{
		"The quick brown fox",
		"jumps over the lazy dog",
		"The dog was lazy",
	}

	results := make(chan SegmentResult, 1)
	var wg sync.WaitGroup

	wg.Add(1)
	go counter.ProcessSegment(1, lines, results, &wg)
	wg.Wait()
	close(results)

	result := <-results

	if result.Error != nil {
		t.Errorf("ProcessSegment() error = %v", result.Error)
	}

	// Check for expected words
	expectedWords := map[string]int{
		"the":   3,
		"dog":   2,
		"lazy":  2,
		"quick": 1,
		"brown": 1,
		"fox":   1,
		"jumps": 1,
		"over":  1,
		"was":   1,
	}

	for word, expectedCount := range expectedWords {
		if count, exists := result.WordCount[word]; !exists {
			t.Errorf("Expected word '%s' not found", word)
		} else if count != expectedCount {
			t.Errorf("Word '%s' count = %d, want %d", word, count, expectedCount)
		}
	}
}

func TestConsolidate(t *testing.T) {
	counter := NewCounter()

	segmentResults := []SegmentResult{
		{
			SegmentID: 1,
			WordCount: WordCount{"the": 2, "quick": 1, "fox": 1},
		},
		{
			SegmentID: 2,
			WordCount: WordCount{"the": 1, "lazy": 2, "dog": 1},
		},
	}

	consolidated := counter.Consolidate(segmentResults)

	expected := WordCount{
		"the":   3,
		"quick": 1,
		"fox":   1,
		"lazy":  2,
		"dog":   1,
	}

	for word, expectedCount := range expected {
		if count, exists := consolidated[word]; !exists {
			t.Errorf("Expected word '%s' not found in consolidated results", word)
		} else if count != expectedCount {
			t.Errorf("Consolidated count for '%s' = %d, want %d", word, count, expectedCount)
		}
	}
}

func TestConcurrentProcessing(t *testing.T) {
	counter := NewCounter()
	lines := make([]string, 100)
	for i := 0; i < 100; i++ {
		lines[i] = "test word concurrent processing example"
	}

	segments := PartitionLines(lines, 4)
	results := make(chan SegmentResult, len(segments))
	var wg sync.WaitGroup

	// Process all segments concurrently
	for i, segment := range segments {
		wg.Add(1)
		go counter.ProcessSegment(i+1, segment, results, &wg)
	}

	wg.Wait()
	close(results)

	// Collect results
	var segmentResults []SegmentResult
	for result := range results {
		if result.Error != nil {
			t.Errorf("Concurrent processing error in segment %d: %v", result.SegmentID, result.Error)
		}
		segmentResults = append(segmentResults, result)
	}

	// Consolidate and verify
	consolidated := counter.Consolidate(segmentResults)

	expectedTotal := 100 * 5 // 100 lines × 5 words per line
	actualTotal := 0
	for _, count := range consolidated {
		actualTotal += count
	}

	if actualTotal != expectedTotal {
		t.Errorf("Total word count = %d, want %d", actualTotal, expectedTotal)
	}
}

func BenchmarkProcessSegment(b *testing.B) {
	counter := NewCounter()
	lines := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		lines[i] = "The quick brown fox jumps over the lazy dog repeatedly in this benchmark test"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results := make(chan SegmentResult, 1)
		var wg sync.WaitGroup
		wg.Add(1)
		go counter.ProcessSegment(1, lines, results, &wg)
		wg.Wait()
		close(results)
		<-results
	}
}
