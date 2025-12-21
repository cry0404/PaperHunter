package ir

import (
	"embed"
	"regexp"
	"strings"
)

//go:embed stopWords.txt
var stopWordsFS embed.FS
var stopWords map[string]bool
type Tokenizer struct {
	stopWords map[string]bool  // 维护一个停用词的集合
}



func NewTokenizer() (*Tokenizer, error) {
	return &Tokenizer {
		stopWords: stopWords,
	}, nil
}

func (t *Tokenizer) Tokenize(text string) []string {
	if text == "" {
		return []string{}
	}

	text = strings.ToLower(text)


	reg := regexp.MustCompile(`[^a-z0-9\s-]`)
	text = reg.ReplaceAllString(text, " ")

	text = strings.ReplaceAll(text, "-", " ")

	words := strings.Fields(text)

	tokens := make([]string, 0, len(words))

	for _, word := range words {
		word = strings.TrimSpace(word)

		if word != "" && len(word) > 1 && !t.stopWords[word] {
			tokens = append(tokens, word)
		}
	}

	return tokens
}

func (t *Tokenizer) TokenizeWithCount(text string) map[string]int {
	result := make(map[string]int)

	if text == "" {
		return result
	}

	tokens := t.Tokenize(text)
	for _, token := range tokens {
		result[token]++
	}

	return result
}