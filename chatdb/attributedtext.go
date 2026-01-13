package chatdb

import (
	"bytes"
	"fmt"
	"unicode/utf8"
)

func extractTextFromAttributedBody(data []byte) (string, error) {
	if len(data) == 0 {
		return "", nil
	}

	text := extractByPatterns(data)
	if text != "" {
		return text, nil
	}

	text = extractTextDirectly(data)
	if text != "" {
		return text, nil
	}

	return "", fmt.Errorf("no text found in attributedBody")
}

func extractByPatterns(data []byte) string {
	patterns := [][]byte{
		[]byte("Karta:"),
		[]byte("Op:"),
		[]byte("Summa:"),
		[]byte("Status:"),
		[]byte("Debitare"),
		[]byte("Suplinire"),
		[]byte("Tranzactie"),
	}

	var bestMatch struct {
		start int
		end   int
		found bool
	}

	for _, pattern := range patterns {
		idx := bytes.Index(data, pattern)
		if idx == -1 {
			continue
		}

		start := idx
		for start > 0 && start > idx-200 {
			if data[start-1] == 0 || data[start-1] < 32 {
				break
			}
			start--
		}

		end := idx + len(pattern)
		for end < len(data) && end < idx+1000 {
			if data[end] == 0 {
				break
			}
			if data[end] < 32 && data[end] != '\n' && data[end] != '\r' && data[end] != '\t' {
				if data[end] < 128 {
					break
				}
			}
			end++
		}

		length := end - start
		if length > (bestMatch.end - bestMatch.start) {
			bestMatch.start = start
			bestMatch.end = end
			bestMatch.found = true
		}
	}

	if bestMatch.found {
		return string(data[bestMatch.start:bestMatch.end])
	}

	return ""
}

func extractTextDirectly(data []byte) string {
	var candidates []string

	i := 0
	for i < len(data) {
		if data[i] < 32 || data[i] >= 127 {
			i++
			continue
		}

		start := i
		validUTF8 := true
		charCount := 0
		lineBreaks := 0

		for i < len(data) {
			if data[i] == 0 {
				break
			}

			if data[i] == '\n' {
				lineBreaks++
			}

			if data[i] < 32 && data[i] != '\n' && data[i] != '\r' && data[i] != '\t' {
				break
			}

			if data[i] >= 127 {
				r, size := utf8.DecodeRune(data[i:])
				if r == utf8.RuneError {
					validUTF8 = false
					break
				}
				i += size
				charCount++
			} else {
				i++
				charCount++
			}
		}

		if validUTF8 && charCount >= 10 {
			text := string(data[start:i])
			if isLikelyMessageText(text) {
				score := len(text) + (lineBreaks * 50)
				candidates = append(candidates, text)
				if score > 100 {
					return text
				}
			}
		}

		i++
	}

	if len(candidates) == 0 {
		return ""
	}

	longestIdx := 0
	for i, candidate := range candidates {
		if len(candidate) > len(candidates[longestIdx]) {
			longestIdx = i
		}
	}

	return candidates[longestIdx]
}

func isLikelyMessageText(s string) bool {
	if len(s) < 10 {
		return false
	}

	if bytes.Contains([]byte(s), []byte("NSMutable")) ||
		bytes.Contains([]byte(s), []byte("NSAttributed")) ||
		bytes.Contains([]byte(s), []byte("NSString")) ||
		bytes.Contains([]byte(s), []byte("NSObject")) ||
		bytes.Contains([]byte(s), []byte("$version")) ||
		bytes.Contains([]byte(s), []byte("$archiver")) ||
		bytes.Contains([]byte(s), []byte("$top")) ||
		bytes.Contains([]byte(s), []byte("$objects")) {
		return false
	}

	alphaCount := 0
	digitCount := 0
	spaceOrPunctCount := 0
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			alphaCount++
		}
		if r >= '0' && r <= '9' {
			digitCount++
		}
		if r == ' ' || r == '.' || r == ',' || r == ':' || r == '\n' {
			spaceOrPunctCount++
		}
	}

	totalMeaningful := alphaCount + digitCount
	ratio := float64(totalMeaningful) / float64(len(s))
	return ratio > 0.4 && spaceOrPunctCount >= 2
}
