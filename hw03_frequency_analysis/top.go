package hw03frequencyanalysis

import (
	"sort"
	"strings"
)

const topSize = 10

func Top10(text string) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	frequencies := make(map[string]int, len(words))
	for _, word := range words {
		frequencies[word]++
	}

	result := make([]string, 0, len(frequencies))
	for word := range frequencies {
		result = append(result, word)
	}

	sort.Slice(result, func(i, j int) bool {
		if frequencies[result[i]] != frequencies[result[j]] {
			return frequencies[result[i]] > frequencies[result[j]]
		}

		return result[i] < result[j]
	})

	if len(result) > topSize {
		result = result[:topSize]
	}

	return result
}
