package ir

import (
	"bufio"
)



func init(){
	loadStopWords()
}


func loadStopWords()  {
	stopWords = make(map[string]bool)
	file, err := stopWordsFS.Open("stopWords.txt")
	if err != nil {
		initDefaultStopWords()
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() { // scan ! EOF
		word := scanner.Text()
		stopWords[word] = true
	}
}

func initDefaultStopWords(){
	stopWords = make(map[string]bool)
	stopMaps := []string{
		"a", "an", "the",
		"and", "or", "but", "nor", "for", "so", "yet",
		"in", "on", "at", "to", "for", "of", "with", "by", "from", "up", "about", "into", "through", "during",
		"i", "you", "he", "she", "it", "we", "they", "this", "that", "these", "those",
		"is", "are", "was", "were", "be", "been", "being",
		"have", "has", "had", "having", "do", "does", "did", "doing", "done",
		"will", "would", "should", "could", "can", "may", "might", "must",
		"as", "if", "than", "then", "when", "where", "why", "how",
		"all", "each", "every", "both", "few", "more", "most", "other", "some", "such", "no", "not", "only", "own", "same", "too", "very",
	}
	
	
	for _, word := range stopMaps {
		stopWords[word] = true
	}
}