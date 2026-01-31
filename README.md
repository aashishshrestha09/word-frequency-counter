# Word Frequency Counter - Multithreaded Implementation

A concurrent word frequency counter built with Go that demonstrates multithreading concepts in operating systems.

## Overview

This program processes text files by partitioning them into N segments and using separate threads (goroutines) to count word frequencies concurrently. After all threads complete, results are consolidated into a final word frequency count.

## Features

- **Concurrent Processing**: Uses Go's goroutines (lightweight threads) for parallel processing
- **Thread-Safe Operations**: Implements proper synchronization with mutexes and wait groups
- **Flexible Segmentation**: Configurable number of segments for processing (byte-range partitioning)
- **Detailed Output**: Shows both intermediate (per-thread) and consolidated results

## Project Structure

```
word-frequency-counter/
├── cmd/
│   └── wordcount/
│       └── main.go          # Main application entry point
├── pkg/
│   └── counter/
│       └── counter.go       # Core word counting logic
├── testdata/
│   └── sample.txt          # Sample text file for testing
├── go.mod                  # Go module definition
└── README.md              # This file
```

## Requirements

- Go 1.21 or higher
- Any operating system (Linux, macOS, Windows)

## Installation

### Option 1: Using Go Install (Recommended)

1. Install Go from https://golang.org/dl/

2. Clone or download this repository

3. Navigate to the project directory:

```bash
cd word-frequency-counter
```

4. Build the program:

```bash
go build -o wordcount ./cmd/wordcount
```

### Option 2: Direct Execution

You can also run directly without building:

```bash
go run ./cmd/wordcount/main.go -file testdata/sample.txt -segments 4 -top 20
```

## Usage

### Basic Usage

```bash
./wordcount -file <path-to-file> -segments <number-of-segments>
```

### Command-Line Arguments

- `-file`: (Required) Path to the text file to process
- `-segments`: Number of segments to partition the file into (default: 4)
- `-verbose`: Show intermediate results from each thread (default: false)
- `-top`: Show top K words in final output (default: 20)
- `-segment-top`: Show top K words per segment when `-verbose` is set (default: 10)
- `-all`: Print all word counts (can be very large)

### Examples

**Example 1: Process with 4 segments (default)**

```bash
./wordcount -file testdata/sample.txt -segments 4
```

**Example 2: Process with 8 segments and verbose output**

```bash
./wordcount -file testdata/sample.txt -segments 8 -verbose
```

**Example 3: Process a custom file**

```bash
./wordcount -file /path/to/your/document.txt -segments 6
```

## Sample Output

```
=== Word Frequency Counter ===
File: testdata/sample.txt
Number of segments: 4

=== Final Consolidated Word Frequencies ===

Total unique words: 87

Total words processed: 167

Top 20 Most Frequent Words:
Rank Word                 Frequency
----------------------------------------
1    the                  15
2    and                  8
3    system               6
4    thread               5
5    operating            5
...
```

## How It Works

1. **File Partitioning**: The program computes N byte ranges from the file size
2. **Boundary Handling**: Each segment reads slightly past its end so words crossing boundaries are not missed
3. **Concurrent Processing**: Each segment is processed by a separate goroutine (thread)
4. **Thread Synchronization**: A WaitGroup ensures the main thread waits for all worker threads
5. **Result Collection**: Each thread sends its word frequency count through a channel
6. **Consolidation**: The main thread merges all segment results into final counts
7. **Output**: Displays both intermediate (optional) and consolidated results

## Thread Safety

The implementation uses several Go concurrency primitives:

- **Goroutines**: Lightweight threads for concurrent segment processing
- **Channels**: For safe communication between threads
- **WaitGroup**: To synchronize thread completion
- **Mutex**: To protect shared data during consolidation

## Testing

To test with the provided sample file:

```bash
# Build first
go build -o wordcount ./cmd/wordcount

# Run with default settings
./wordcount -file testdata/sample.txt -segments 4 -verbose
```

## Performance Considerations

- More segments don't always mean better performance due to overhead
- Optimal segment count typically matches the number of CPU cores
- For small files, overhead may exceed benefits of parallelization
- Go's scheduler efficiently maps goroutines to OS threads
