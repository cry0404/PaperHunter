package ir

import (
	"reflect"
	"testing"
)

func TestNewTokenizer(t *testing.T) {
	tokenizer, err := NewTokenizer()
	if err != nil {
		t.Fatalf("NewTokenizer() failed: %v", err)
	}

	if tokenizer == nil {
		t.Fatal("NewTokenizer() returned nil")
	}

	if tokenizer.stopWords == nil {
		t.Fatal("stopWords map not initialized")
	}
}

func TestTokenizer_Tokenize(t *testing.T) {
	tokenizer, _ := NewTokenizer()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "simple words",
			input:    "Hello world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "with punctuation",
			input:    "Hello, world! This is a test.",
			expected: []string{"hello", "world", "test"},
		},
		{
			name:     "with stop words",
			input:    "This is a test with stop words",
			expected: []string{"test", "stop", "words"},
		},
		{
			name:     "with hyphens",
			input:    "deep-learning neural-networks",
			expected: []string{"deep", "learning", "neural", "networks"},
		},
		{
			name:     "single character words",
			input:    "I am a AI",
			expected: []string{"am", "ai"},
		},
		{
			name:     "numbers and text",
			input:    "GPT-4 is amazing 2023",
			expected: []string{"gpt", "amazing", "2023"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenizer.Tokenize(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Tokenize() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestTokenizer_TokenizeWithCount(t *testing.T) {
	tokenizer, _ := NewTokenizer()

	tests := []struct {
		name     string
		input    string
		expected map[string]int
	}{
		{
			name:     "empty string",
			input:    "",
			expected: map[string]int{},
		},
		{
			name:     "simple words",
			input:    "hello world hello",
			expected: map[string]int{"hello": 2, "world": 1},
		},
		{
			name:     "with stop words",
			input:    "this is a test test",
			expected: map[string]int{"test": 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenizer.TokenizeWithCount(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("TokenizeWithCount() = %v, expected %v", result, tt.expected)
			}
		})
	}
}