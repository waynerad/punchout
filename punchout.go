package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type splitEntryRecord struct {
	kind    int
	section string
	length  int
}

func intToStr(ii int) string {
	return strconv.FormatInt(int64(ii), 10)
}

func readline(r *bufio.Reader) (string, error) {
	var ln []byte
	line, isPrefix, err := r.ReadLine()
	ln = append(ln, line...)
	for isPrefix && (err == nil) {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}
	return string(ln), err
}

// eosMarkers := []string{". ", " ! ", " ? ", "EOS"}
func findFirstEndOfSentence(str string, eosMarkers []string) (int, string) {
	ls := len(str)
	first := -1
	found := ""
	for _, marker := range eosMarkers {
		i := strings.Index(str, marker)
		if first < 0 {
			first = i
			found = marker
		} else {
			if i >= 0 {
				if i < first {
					first = i
					found = marker
				}
			}
		}
		if first < 0 {
			lm := len(marker)
			if marker[lm-1:lm] == " " {
				if str[(ls-lm+1):ls] == marker[0:lm-1] {
					first = ls - lm + 1
					found = marker
				}
			}
		}
	}
	return first, found
}

func smartSplitter(sentence string) []splitEntryRecord {
	rv := make([]splitEntryRecord, 0)
	inknd := 0
	current := ""
	punctuationChars := " .,:;!?«»()" // should apostrophes and dashes go here?
	punctuationChars = punctuationChars + string(34)
	count := 0
	var entry splitEntryRecord
	for _, chSentence := range sentence {
		neuknd := 1
		for _, chPunct := range punctuationChars {
			if chSentence == chPunct {
				neuknd = 2
			}
		}
		if neuknd != inknd {
			entry.kind = inknd
			entry.section = current
			entry.length = count
			if inknd != 0 {
				rv = append(rv, entry)
			}
			current = ""
			inknd = neuknd
		}
		current = current + string(chSentence)
		count++
	}
	entry.kind = inknd
	entry.section = current
	entry.length = count
	rv = append(rv, entry)
	return rv
}

func blankizeSentence(sentence string, fhOut *os.File, alreadyused map[string]bool, alreadyasked map[string]bool, ignoreRepeats bool) {
	fmt.Fprintln(fhOut, "")
	fmt.Fprintln(fhOut, "# "+sentence)
	splitSen := smartSplitter(sentence)
	for i := 0; i < len(splitSen); i++ {
		if splitSen[i].kind == 1 {
			blankSen := ""
			inblankchars := 0
			outofblankchars := 0
			combinedAnswers := ""
			combinedQuestions := ""
			for j := 0; j < len(splitSen); j++ {
				if j == i {
					blankSen = blankSen + "_" + splitSen[j].section + "_"
					inblankchars += splitSen[j].length
					combinedAnswers = combinedAnswers + "_" + splitSen[j].section
				} else {
					blankSen = blankSen + splitSen[j].section
					outofblankchars += splitSen[j].length
					combinedQuestions = combinedQuestions + "_" + splitSen[j].section
				}
			}
			fmt.Fprintln(fhOut, blankSen)
			if outofblankchars == 0 {
				fmt.Fprintln(fhOut, "# issue: ^^ All characters are in blank!")
			}
			if inblankchars == 0 {
				fmt.Fprintln(fhOut, "# issue: ^^ No characters are in blanks! Nothing is being asked!")
			}

			_, ok := alreadyused[combinedAnswers]
			if ok {
				if !ignoreRepeats {
					fmt.Fprintln(fhOut, "# issue: ^^ answer is a repetition")
				}
			} else {
				alreadyused[combinedAnswers] = true
			}

			_, ok = alreadyasked[combinedQuestions]
			if ok {
				fmt.Fprintln(fhOut, "# issue: ^^ question being asked is indistinguishable from a previous question!")
			} else {
				alreadyasked[combinedQuestions] = true
			}

		}
	}
}

func makeblanks(infile string, outfile string) {
	fhOut, err := os.Create(outfile)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Could not open output file:", outfile)
		return
	}
	defer fhOut.Close()
	fhOriginal, err := os.Open(infile)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Not found:", infile)
		return
	}
	defer fhOriginal.Close()
	fhIn := bufio.NewReader(fhOriginal)
	alreadyused := make(map[string]bool)
	alreadyasked := make(map[string]bool)
	alreadySentence := make(map[string]bool)
	chapterNumber := 1
	lineCount := 0
	inLineNum := 0
	eosMarkers := make([]string, 0)
	eosMarkers = append(eosMarkers, "EOS")

	treatEndOfLineAsEndOfSentence := false
	outputSourceLineNumbers := false
	ignoreRepeats := false
	chapterBreakLines := 0

	keepgoing := true
	for keepgoing {
		inline, err := fhIn.ReadString('\n')
		inLineNum++
		if err == nil {
			inline = strings.Trim(inline, "\r\n\t ")
			if len(inline) == 0 {
				fmt.Fprintln(fhOut, inline) // retain blank lines
			} else {
				if inline[0:1] == "#" {
					fmt.Fprintln(fhOut, inline) // output comments with no modification
					if inline == "#reset" {
						// throw away our map that keeps track of repetitions
						alreadyused = make(map[string]bool)
						fmt.Fprintln(fhOut, "# issue: reset repetitions tracker")
					}
					if inline == "#lines" {
						treatEndOfLineAsEndOfSentence = true
					}
					if inline == "#sourcelinenumbers" {
						outputSourceLineNumbers = true
					}
					if inline == "#commas" {
						eosMarkers = append(eosMarkers, ", ")
						eosMarkers = append(eosMarkers, ". ")
						eosMarkers = append(eosMarkers, " ! ")
						eosMarkers = append(eosMarkers, " ? ")
					}
					if inline == "#punctnocommas" {
						eosMarkers = append(eosMarkers, ". ")
						eosMarkers = append(eosMarkers, " ! ")
						eosMarkers = append(eosMarkers, " ? ")
					}
					if inline == "#ignorerepeats" {
						ignoreRepeats = true
					}
					// "#chapterbreaks "
					//  123456789012345
					//  0123456789012345
					if len(inline) > 15 {
						if inline[:15] == "#chapterbreaks " {
							tempVar, err := strconv.ParseInt(inline[15:], 10, 0)
							if err == nil {
								chapterBreakLines = int(tempVar)
								chapterNumber = 1
								fmt.Fprintln(fhOut, "")
								fmt.Fprintln(fhOut, "# chapter "+intToStr(chapterNumber))
							}
						}
					}
				} else {
					if treatEndOfLineAsEndOfSentence {
						inline = inline + "EOS"
					}
					first, found := findFirstEndOfSentence(inline, eosMarkers)
					for first > 0 {
						var item string
						if found == "EOS" {
							item = inline[0:first]
						} else {
							item = inline[0:first] + found
						}
						if (first + len(found)) >= len(inline) {
							inline = ""
							first = -1
						} else {
							inline = inline[first+len(found):]
							first, found = findFirstEndOfSentence(inline, eosMarkers)
						}
						_, ok := alreadySentence[item]
						if !ok {
							if outputSourceLineNumbers {
								fmt.Fprintln(fhOut, "# line "+intToStr(inLineNum))
							}
							blankizeSentence(item, fhOut, alreadyused, alreadyasked, ignoreRepeats)
							alreadySentence[item] = true
							lineCount++
							if lineCount == chapterBreakLines {
								chapterNumber++
								fmt.Fprintln(fhOut, "")
								fmt.Fprintln(fhOut, "# chapter "+intToStr(chapterNumber))
								lineCount = 0
							}
						}
					}
				}
			}
		} else {
			if err != io.EOF {
				fmt.Println(err)
			}
			keepgoing = false
		}
	}
}

func makeBlanksFromBaseName(basename string) {
	fmt.Println(basename)
	infile := basename + "-original.txt"
	outfile := basename + "-blanks.txt"
	makeblanks(infile, outfile)
}

func main() {
	flag.Parse()
	for i := 0; i < flag.NArg(); i++ {
		basename := flag.Arg(i)
		makeBlanksFromBaseName(basename)
	}
}
