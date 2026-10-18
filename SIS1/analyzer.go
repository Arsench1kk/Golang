package main

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var ErrEmptyInput = errors.New("text input must not be empty")

type AnalysisResult struct {
	Words       int
	Runes       int
	Bytes       int
	UniqueWords int
}

type Processor interface {
	Process(input string) (AnalysisResult, error)
}

type TextAnalyzer struct{}

var _ Processor = TextAnalyzer{}

func (TextAnalyzer) Process(input string) (AnalysisResult, error) {
	if strings.TrimSpace(input) == "" {
		return AnalysisResult{}, ErrEmptyInput
	}

	words := strings.Fields(input)
	uniqueWords := make(map[string]struct{}, len(words))
	for _, word := range words {
		uniqueWords[word] = struct{}{}
	}

	return AnalysisResult{
		Words:       len(words),
		Runes:       utf8.RuneCountInString(input),
		Bytes:       len(input),
		UniqueWords: len(uniqueWords),
	}, nil
}
