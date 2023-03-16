package wordsplit

import (
	"bufio"
	"os"
	"strings"
	"unicode"
)

type WordsDB struct {
	words      map[string]struct{}
	wordLoadCh chan string
}

func New() WordsDB {
	db := WordsDB{
		words:      map[string]struct{}{},
		wordLoadCh: make(chan string),
	}

	go func() {
		for word := range db.wordLoadCh {
			db.words[strings.ToLower(word)] = struct{}{}
		}
	}()
	return db
}

func (db WordsDB) Contains(word string) bool {
	if word == "" {
		return false
	}
	_, ok := db.words[strings.ToLower(word)]
	return ok
}

func (db WordsDB) Add(word string) {
	db.wordLoadCh <- word
}

func (db WordsDB) LoadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		db.Add(word)
	}
	return nil
}

type StringRange struct {
	Start, End int
}

func (r StringRange) Slice(input []rune) []rune {
	return input[r.Start:r.End]
}

func (r StringRange) Len() int {
	return r.End - r.Start
}

type SplitSequence []StringRange

func (db WordsDB) splitAsync(input []rune, start int, minWordLength int, maxNonWordLength int, sequenceOutputCh chan<- SplitSequence) {
	// Recursion base case: splitting an empty string
	if start >= len(input) {
		close(sequenceOutputCh)
		return
	}

	// If we start with a non-alnum rune, skip it
	if !(unicode.IsLetter(input[start]) || unicode.IsDigit(input[start])) {
		db.splitAsync(input, start+1, minWordLength, maxNonWordLength, sequenceOutputCh)
		return
	}

	// Recursion base case: no split happens, just process the whole input as a single "word"
	noSplitRange := StringRange{start, len(input)}
	noSplitIsWord := db.Contains(string(noSplitRange.Slice(input)))
	if (noSplitIsWord && len(input)-start >= minWordLength) ||
		(len(input)-start <= maxNonWordLength) {
		sequenceOutputCh <- SplitSequence{noSplitRange}
	}

	// Remember where we split (for merges) to avoid duplicate results
	alreadyOutputRanges := map[StringRange]struct{}{}

	// Consider splitting on every range in the remaining input
	for end := start + 1; end <= len(input); end++ {
		if !(unicode.IsLetter(input[end-1]) || unicode.IsDigit(input[end-1])) {
			// If the last character is not alphanumeric, stop building
			break
		}

		currRange := StringRange{start, end}
		nextIsAlnum := end+1 < len(input) && unicode.In(input[end+1], unicode.Letter, unicode.Digit)
		isWord := db.Contains(string(currRange.Slice(input)))

		rangeSkipped := false
		rangeSkipped = rangeSkipped || !nextIsAlnum
		if isWord {
			rangeSkipped = rangeSkipped || end-start < minWordLength
		} else {
			rangeSkipped = rangeSkipped || end-start > maxNonWordLength
		}
		if rangeSkipped {
			continue
		}

		// A valid word or non-word, we can split on it
		// Try to split everything after the head
		subsequenceCh := make(chan SplitSequence)
		go db.splitAsync(input, end, minWordLength, maxNonWordLength, subsequenceCh)

		// Take each split subsequence and prepend the current chunk to it
		for subsequence := range subsequenceCh {
			var nextRange *StringRange
			var nextRangeIsNonWord bool
			if len(subsequence) > 0 {
				nextRange = &subsequence[0]
				nextRangeIsNonWord = !db.Contains(string(nextRange.Slice(input)))
			}

			// Combine the current range with the split sequence of ranges from the recursive call
			var newSequence SplitSequence
			if !isWord && nextRangeIsNonWord {
				// Merge the current nonword and the next; if the result is too long, skip this subsequence
				if currRange.Len()+nextRange.Len() > maxNonWordLength {
					continue
				}
				mergedRange := StringRange{currRange.Start, nextRange.End}

				if _, ok := alreadyOutputRanges[mergedRange]; ok {
					// We already output this one!
					continue
				}
				alreadyOutputRanges[mergedRange] = struct{}{}

				// Prepend the merged range to the remaining part of the subsequence
				newSequence = append(newSequence, mergedRange)
				newSequence = append(newSequence, subsequence[1:]...)
			} else {
				// If we don't have two adjacent nonwords, simply prepend the current range to the subsequence
				alreadyOutputRanges[currRange] = struct{}{}
				newSequence = append(newSequence, currRange)
				newSequence = append(newSequence, subsequence...)
			}

			// Async output the new sequence (containing the current range)
			sequenceOutputCh <- newSequence
		}
	}

	close(sequenceOutputCh) // Close the output channel to indicate being done
}

func (db WordsDB) Split(input string, minWordLength int, maxNonWordLength int) (outputSplits []SplitSequence) {
	sequencesCh := make(chan SplitSequence)
	go db.splitAsync([]rune(input), 0, minWordLength, maxNonWordLength, sequencesCh)

	for seq := range sequencesCh {
		// Sequence is OK, add it to outputs
		outputSplits = append(outputSplits, seq)
	}
	return
}
