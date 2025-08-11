package console

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

// captureOutput captures stdout during test execution.
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestPrint(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected string
	}{
		{
			name:     "single string",
			args:     []interface{}{"hello"},
			expected: "hello",
		},
		{
			name:     "multiple strings",
			args:     []interface{}{"hello", " ", "world"},
			expected: "hello world",
		},
		{
			name:     "mixed types",
			args:     []interface{}{"count:", 42, " items"},
			expected: "count:42 items",
		},
		{
			name:     "empty args",
			args:     []interface{}{},
			expected: "",
		},
		{
			name:     "single integer",
			args:     []interface{}{123},
			expected: "123",
		},
		{
			name:     "boolean value",
			args:     []interface{}{true},
			expected: "true",
		},
		{
			name:     "nil value",
			args:     []interface{}{nil},
			expected: "<nil>",
		},
		{
			name:     "slice value",
			args:     []interface{}{[]int{1, 2, 3}},
			expected: "[1 2 3]",
		},
		{
			name:     "struct value",
			args:     []interface{}{struct{ Name string }{Name: "test"}},
			expected: "{test}",
		},
		{
			name:     "float value",
			args:     []interface{}{3.14},
			expected: "3.14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				Print(tt.args...)
			})

			if output != tt.expected {
				t.Errorf("Print() = %q, want %q", output, tt.expected)
			}
		})
	}
}

func TestPrintln(t *testing.T) {
	tests := []struct {
		name     string
		args     []interface{}
		expected string
	}{
		{
			name:     "single string",
			args:     []interface{}{"hello"},
			expected: "hello\n",
		},
		{
			name:     "multiple strings",
			args:     []interface{}{"hello", "world"},
			expected: "hello world\n",
		},
		{
			name:     "mixed types",
			args:     []interface{}{"count:", 42},
			expected: "count: 42\n",
		},
		{
			name:     "empty args",
			args:     []interface{}{},
			expected: "\n",
		},
		{
			name:     "single integer",
			args:     []interface{}{123},
			expected: "123\n",
		},
		{
			name:     "boolean value",
			args:     []interface{}{false},
			expected: "false\n",
		},
		{
			name:     "multiple integers",
			args:     []interface{}{1, 2, 3},
			expected: "1 2 3\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				Println(tt.args...)
			})

			if output != tt.expected {
				t.Errorf("Println() = %q, want %q", output, tt.expected)
			}
		})
	}
}

func TestPrintf(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		expected string
	}{
		{
			name:     "simple string format",
			format:   "Hello %s",
			args:     []interface{}{"world"},
			expected: "Hello world",
		},
		{
			name:     "integer format",
			format:   "Number: %d",
			args:     []interface{}{42},
			expected: "Number: 42",
		},
		{
			name:     "multiple values",
			format:   "Name: %s, Age: %d",
			args:     []interface{}{"John", 25},
			expected: "Name: John, Age: 25",
		},
		{
			name:     "float format",
			format:   "Pi: %.2f",
			args:     []interface{}{3.14159},
			expected: "Pi: 3.14",
		},
		{
			name:     "no format specifiers",
			format:   "static text",
			args:     []interface{}{},
			expected: "static text",
		},
		{
			name:     "empty format",
			format:   "",
			args:     []interface{}{},
			expected: "",
		},
		{
			name:     "boolean format",
			format:   "Status: %t",
			args:     []interface{}{true},
			expected: "Status: true",
		},
		{
			name:     "hex format",
			format:   "Hex: %x",
			args:     []interface{}{255},
			expected: "Hex: ff",
		},
		{
			name:     "quoted string format",
			format:   "Quoted: %q",
			args:     []interface{}{"hello world"},
			expected: "Quoted: \"hello world\"",
		},
		{
			name:     "pointer format",
			format:   "Value: %v",
			args:     []interface{}{&struct{ Name string }{Name: "test"}},
			expected: "Value: &{test}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				Printf(tt.format, tt.args...)
			})

			if output != tt.expected {
				t.Errorf("Printf() = %q, want %q", output, tt.expected)
			}
		})
	}
}

func TestConcurrentPrint(t *testing.T) {
	const numGoroutines = 10
	const messagesPerGoroutine = 5

	var wg sync.WaitGroup

	// Capture all output from concurrent operations
	allOutput := captureOutput(func() {
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := 0; j < messagesPerGoroutine; j++ {
					Print(fmt.Sprintf("goroutine-%d-msg-%d ", id, j))
				}
			}(i)
		}
		wg.Wait()
	})

	// Verify that all expected messages appear in the output (may be interleaved)
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < messagesPerGoroutine; j++ {
			expectedMsg := fmt.Sprintf("goroutine-%d-msg-%d ", i, j)
			if !strings.Contains(allOutput, expectedMsg) {
				t.Errorf("Missing expected message: %q in output: %q", expectedMsg, allOutput)
			}
		}
	}
}

func TestConcurrentPrintln(t *testing.T) {
	const numGoroutines = 5
	const messagesPerGoroutine = 3

	var wg sync.WaitGroup

	// Capture all output from concurrent operations
	allOutput := captureOutput(func() {
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := 0; j < messagesPerGoroutine; j++ {
					Println(fmt.Sprintf("goroutine-%d-line-%d", id, j))
				}
			}(i)
		}
		wg.Wait()
	})

	// Verify that all expected messages appear in the output (may be interleaved)
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < messagesPerGoroutine; j++ {
			expectedMsg := fmt.Sprintf("goroutine-%d-line-%d\n", i, j)
			if !strings.Contains(allOutput, expectedMsg) {
				t.Errorf("Missing expected message: %q in output: %q", expectedMsg, allOutput)
			}
		}
	}
}

func TestConcurrentPrintf(t *testing.T) {
	const numGoroutines = 5
	const messagesPerGoroutine = 3

	var wg sync.WaitGroup

	// Capture all output from concurrent operations
	allOutput := captureOutput(func() {
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := 0; j < messagesPerGoroutine; j++ {
					Printf("goroutine-%d-formatted-%d ", id, j)
				}
			}(i)
		}
		wg.Wait()
	})

	// Verify that all expected messages appear in the output (may be interleaved)
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < messagesPerGoroutine; j++ {
			expectedMsg := fmt.Sprintf("goroutine-%d-formatted-%d ", i, j)
			if !strings.Contains(allOutput, expectedMsg) {
				t.Errorf("Missing expected message: %q in output: %q", expectedMsg, allOutput)
			}
		}
	}
}

// Test edge cases and special scenarios.
func TestPrintSpecialCharacters(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "newline character",
			input:    "line1\nline2",
			expected: "line1\nline2",
		},
		{
			name:     "tab character",
			input:    "col1\tcol2",
			expected: "col1\tcol2",
		},
		{
			name:     "unicode characters",
			input:    "Hello 世界",
			expected: "Hello 世界",
		},
		{
			name:     "emoji",
			input:    "Status: ✅",
			expected: "Status: ✅",
		},
		{
			name:     "special symbols",
			input:    "Price: $100.50",
			expected: "Price: $100.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				Print(tt.input)
			})

			if output != tt.expected {
				t.Errorf("Print() = %q, want %q", output, tt.expected)
			}
		})
	}
}

func TestPrintfFormatErrors(t *testing.T) {
	// Test Printf with mismatched format specifiers
	// Note: Go's fmt.Printf handles these gracefully
	tests := []struct {
		name     string
		format   string
		args     []interface{}
		contains string // What the output should contain
	}{
		{
			name:     "too few arguments",
			format:   "Name: %s, Age: %d",
			args:     []interface{}{"John"},
			contains: "Name: John, Age: %!d(MISSING)",
		},
		{
			name:     "too many arguments",
			format:   "Name: %s",
			args:     []interface{}{"John", 25},
			contains: "Name: John%!(EXTRA int=25)",
		},
		{
			name:     "wrong type",
			format:   "Number: %d",
			args:     []interface{}{"not a number"},
			contains: "Number: %!d(string=not a number)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				Printf(tt.format, tt.args...)
			})

			if output != tt.contains {
				t.Errorf("Printf() = %q, want %q", output, tt.contains)
			}
		})
	}
}

// Benchmark tests.
func BenchmarkPrint(b *testing.B) {
	// Redirect output to discard during benchmarking
	old := os.Stdout
	os.Stdout = nil
	defer func() { os.Stdout = old }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Print("benchmark test message")
	}
}

func BenchmarkPrintln(b *testing.B) {
	// Redirect output to discard during benchmarking
	old := os.Stdout
	os.Stdout = nil
	defer func() { os.Stdout = old }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Println("benchmark test message")
	}
}

func BenchmarkPrintf(b *testing.B) {
	// Redirect output to discard during benchmarking
	old := os.Stdout
	os.Stdout = nil
	defer func() { os.Stdout = old }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Printf("benchmark test message %d", i)
	}
}

func BenchmarkPrintConcurrent(b *testing.B) {
	// Redirect output to discard during benchmarking
	old := os.Stdout
	os.Stdout = nil
	defer func() { os.Stdout = old }()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Print("concurrent benchmark")
		}
	})
}

// Test large output.
func TestPrintLargeOutput(t *testing.T) {
	largeString := string(make([]byte, 10000))
	for i := range largeString {
		largeString = largeString[:i] + "A" + largeString[i+1:]
	}

	output := captureOutput(func() {
		Print(largeString)
	})

	if len(output) != len(largeString) {
		t.Errorf("Large output length = %d, want %d", len(output), len(largeString))
	}

	if output != largeString {
		t.Error("Large output content mismatch")
	}
}

func TestPrintWithNilSlice(t *testing.T) {
	var nilSlice []string
	output := captureOutput(func() {
		Print(nilSlice)
	})

	expected := "[]"
	if output != expected {
		t.Errorf("Print(nil slice) = %q, want %q", output, expected)
	}
}

func TestPrintlnWithNilInterface(t *testing.T) {
	var nilInterface interface{}
	output := captureOutput(func() {
		Println(nilInterface)
	})

	expected := "<nil>\n"
	if output != expected {
		t.Errorf("Println(nil interface) = %q, want %q", output, expected)
	}
}
