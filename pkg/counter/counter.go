package counter

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// WordCount represents the frequency count of words
type WordCount map[string]int

// SegmentResult holds the result from processing a file segment
type SegmentResult struct {
	SegmentID int
	StartByte int64
	EndByte   int64 // exclusive end of the owned range
	WordCount WordCount
	Error     error
}

// Counter manages concurrent word frequency counting
type Counter struct {
	mu           sync.Mutex
	consolidated WordCount
	wordPattern  *regexp.Regexp
}

// NewCounter creates a new Counter instance
func NewCounter() *Counter {
	return &Counter{
		consolidated: make(WordCount),
		wordPattern:  regexp.MustCompile(`[a-zA-Z]+`),
	}
}

type FileSegment struct {
	ID      int
	Start   int64
	End     int64 // exclusive end of owned range
	ReadEnd int64 // exclusive end to read (End + overlap)
}

// CountFileConcurrently partitions the file into N byte segments and counts words concurrently.
//
// Segments are defined by byte ranges (not line ranges). To avoid missing words that cross a
// segment boundary, each segment (except the last) reads past its owned End by an overlap
// window and only counts words whose start offset is within [Start, End).
func CountFileConcurrently(filePath string, numSegments int) ([]SegmentResult, WordCount, error) {
	if numSegments < 1 {
		return nil, nil, fmt.Errorf("segments must be >= 1")
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("stat file: %w", err)
	}
	if st.Size() == 0 {
		return []SegmentResult{}, make(WordCount), nil
	}

	segments := partitionFileByBytes(st.Size(), numSegments, 64*1024)
	resultsCh := make(chan SegmentResult, len(segments))
	var wg sync.WaitGroup

	for _, seg := range segments {
		seg := seg
		r := io.NewSectionReader(f, seg.Start, seg.ReadEnd-seg.Start)
		wg.Add(1)
		go func() {
			defer wg.Done()
			wc, err := countWordsInOwnedRange(r, seg.Start, seg.End)
			resultsCh <- SegmentResult{
				SegmentID: seg.ID,
				StartByte: seg.Start,
				EndByte:   seg.End,
				WordCount: wc,
				Error:     err,
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	segmentResults := make([]SegmentResult, 0, len(segments))
	for res := range resultsCh {
		if res.Error != nil {
			return nil, nil, fmt.Errorf("segment %d: %w", res.SegmentID, res.Error)
		}
		segmentResults = append(segmentResults, res)
	}

	sort.Slice(segmentResults, func(i, j int) bool {
		return segmentResults[i].SegmentID < segmentResults[j].SegmentID
	})

	consolidated := make(WordCount)
	for _, res := range segmentResults {
		for w, c := range res.WordCount {
			consolidated[w] += c
		}
	}

	return segmentResults, consolidated, nil
}

func partitionFileByBytes(fileSize int64, numSegments int, overlapBytes int64) []FileSegment {
	if numSegments < 1 {
		numSegments = 1
	}
	if fileSize <= 0 {
		return []FileSegment{}
	}

	// If the file is very small, avoid creating empty segments.
	if int64(numSegments) > fileSize {
		numSegments = max(int(fileSize), 1)
	}

	segments := make([]FileSegment, 0, numSegments)
	for i := 0; i < numSegments; i++ {
		start := (int64(i) * fileSize) / int64(numSegments)
		end := (int64(i+1) * fileSize) / int64(numSegments)
		readEnd := end
		if i != numSegments-1 {
			readEnd = min(end+overlapBytes, fileSize)
		}

		segments = append(segments, FileSegment{
			ID:      i + 1,
			Start:   start,
			End:     end,
			ReadEnd: readEnd,
		})
	}

	return segments
}

func countWordsInOwnedRange(r io.Reader, absoluteStart int64, ownedEnd int64) (WordCount, error) {
	const bufSize = 32 * 1024
	buf := make([]byte, bufSize)

	wc := make(WordCount)
	var (
		absOffset  = absoluteStart
		inWord     bool
		wordStart  int64
		wordBuffer []byte
	)

	flush := func() {
		if !inWord {
			return
		}
		// Only count the word if it started inside this segment's owned range.
		if wordStart < ownedEnd {
			for i := range wordBuffer {
				b := wordBuffer[i]
				if b >= 'A' && b <= 'Z' {
					wordBuffer[i] = b + ('a' - 'A')
				}
			}
			wc[string(wordBuffer)]++
		}
		inWord = false
		wordBuffer = wordBuffer[:0]
	}

	for {
		n, err := r.Read(buf)
		if n > 0 {
			for i := 0; i < n; i++ {
				b := buf[i]
				isLetter := (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
				if isLetter {
					if !inWord {
						inWord = true
						wordStart = absOffset
						wordBuffer = wordBuffer[:0]
					}
					wordBuffer = append(wordBuffer, b)
				} else {
					flush()
				}
				absOffset++
			}
		}

		if err != nil {
			if err == io.EOF {
				flush()
				return wc, nil
			}
			return nil, err
		}
	}
}

// ProcessSegment processes a segment of text and returns word frequencies
func (c *Counter) ProcessSegment(segmentID int, lines []string, results chan<- SegmentResult, wg *sync.WaitGroup) {
	defer wg.Done()

	localCount := make(WordCount)

	for _, line := range lines {
		words := c.extractWords(line)
		for _, word := range words {
			// Normalize to lowercase for case-insensitive counting
			word = strings.ToLower(word)
			localCount[word]++
		}
	}

	results <- SegmentResult{
		SegmentID: segmentID,
		WordCount: localCount,
		Error:     nil,
	}
}

// extractWords extracts valid words from a line of text
func (c *Counter) extractWords(line string) []string {
	return c.wordPattern.FindAllString(line, -1)
}

// Consolidate merges all segment results into a final word frequency count
func (c *Counter) Consolidate(segmentResults []SegmentResult) WordCount {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.consolidated = make(WordCount)

	for _, result := range segmentResults {
		for word, count := range result.WordCount {
			c.consolidated[word] += count
		}
	}

	return c.consolidated
}

// GetConsolidated returns the consolidated word count (thread-safe)
func (c *Counter) GetConsolidated() WordCount {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Return a copy to prevent external modification
	result := make(WordCount, len(c.consolidated))
	for k, v := range c.consolidated {
		result[k] = v
	}
	return result
}

// PartitionLines divides lines into N roughly equal segments
func PartitionLines(lines []string, numSegments int) [][]string {
	if numSegments <= 0 {
		numSegments = 1
	}
	if len(lines) == 0 {
		return [][]string{}
	}
	if numSegments > len(lines) {
		numSegments = len(lines)
	}

	segments := make([][]string, numSegments)
	linesPerSegment := len(lines) / numSegments
	remainder := len(lines) % numSegments

	startIdx := 0
	for i := 0; i < numSegments; i++ {
		endIdx := startIdx + linesPerSegment
		if i < remainder {
			endIdx++
		}
		segments[i] = lines[startIdx:endIdx]
		startIdx = endIdx
	}

	return segments
}

// ReadLines reads all lines from a file
func ReadLines(scanner *bufio.Scanner) []string {
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}
