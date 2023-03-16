package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fsufitch/wordsplit"
)

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

	db := wordsplit.New()
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
		sequences := db.Split(input, *minWordLengthFlag, *maxNonWordLengthFlag)
		if len(sequences) == 0 {
			fmt.Printf("%s ???\n", input)
		}
		for _, sequence := range sequences {
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
