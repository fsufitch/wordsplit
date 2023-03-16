package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
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

func (r StringRange) Slice(input string) string {
	return input[r.Start:r.End]
}

func (r StringRange) Len() int {
	return r.End - r.Start
}

type SplitSequence []StringRange

func (db WordsDB) isValidRange(input string, sRange StringRange, minWordLength int, maxNonWordLength int) bool {
	if db.Contains(sRange.Slice(input)) {
		if sRange.Len() >= minWordLength {
			return true
		}
	} else if sRange.Len() <= maxNonWordLength {
		return true
	}
	return false
}

func (db WordsDB) splitAsync(input string, start int, minWordLength int, maxNonWordLength int, sequenceOutputCh chan<- SplitSequence) {
	// Recursion base case: splitting an empty string
	if start >= len(input) {
		close(sequenceOutputCh)
		return
	}

	// Recursion base case: no split happens, just process the whole input as
	noSplitRange := StringRange{start, len(input)}
	if db.isValidRange(input, noSplitRange, minWordLength, maxNonWordLength) {
		sequenceOutputCh <- SplitSequence{noSplitRange}
	}

	// Remember where we split (for merges) to avoid duplicate results
	alreadyOutputRanges := map[StringRange]struct{}{}

	// Consider splitting on every range in the remaining input
	for end := start + 1; end < len(input); end++ {
		currRange := StringRange{start, end}
		isWord := db.Contains(currRange.Slice(input))

		if isWord {
			if currRange.Len() < minWordLength {
				// currRange has a word, but it's too short; skip it
				continue
			}

		} else {
			if currRange.Len() > maxNonWordLength {
				// currRange has a nonword, but it's too long; skip it
				continue
			}
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
				nextRangeIsNonWord = !db.Contains(nextRange.Slice(input))
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
	go db.splitAsync(input, 0, minWordLength, maxNonWordLength, sequencesCh)

	for seq := range sequencesCh {
		// Sequence is OK, add it to outputs
		outputSplits = append(outputSplits, seq)
	}
	return
}

func main() {
	minWordLengthFlag := flag.Int("w", 3, "minimum valid word length")
	maxNonWordLengthFlag := flag.Int("nw", 3, "maximum valid nonword length")
	wordsFileFlag := flag.String("f", os.Getenv("WORDS_FILE"), "words file to use; default from WORDS_FILE")
	flag.Parse()

	if *wordsFileFlag == "" {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "No words file given; set WORDS_FILE or use -f")
		os.Exit(1)
	}

	db := New()
	if err := db.LoadFile(*wordsFileFlag); err != nil {
		err = fmt.Errorf("failed to load words file: %w", err)
		panic(err)
	}

	inputWordsCh := make(chan string)

	go func() {
		// Async read input in case stdin is huge
		if flag.NArg() == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Split(bufio.ScanWords)
			for scanner.Scan() {
				inputWordsCh <- strings.TrimSpace(scanner.Text())
			}
		} else {
			for _, arg := range flag.Args() {
				inputWordsCh <- strings.TrimSpace(arg)
			}
		}
		close(inputWordsCh)
	}()

	for input := range inputWordsCh {
		for _, sequence := range db.Split(input, *minWordLengthFlag, *maxNonWordLengthFlag) {
			outArray := []string{}
			for _, chunk := range sequence {
				word := input[chunk.Start:chunk.End]
				if db.Contains(word) {
					outArray = append(outArray, word)
				} else {
					outArray = append(outArray, "("+word+")")
				}

			}
			byts, _ := json.Marshal(outArray)
			fmt.Printf("%s -> %s\n", input, string(byts))
		}
	}

}
