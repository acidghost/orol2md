// (c) Copyright 2021, orol2md Authors.
//
// Licensed under the terms of the GNU GPL License version 3.

package main

import (
	"bufio"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"
)

var (
	flagOutput     = flag.String("o", "", "output folder (default to current)")
	flagTitle      = flag.String("s", "", "search term; required")
	flagMultiAllow = flag.Bool("f", false, "allow multiple matches w/o asking")
	flagObsidian   = flag.Bool("obs", false, "obsidian mode")
)

const BookNotesTmpl = `# {{.Title}}
- Authors: {{.Authors}}
- URL: {{.URL}}

{{- range .Chapters}}

## {{.Title}}
- URL: {{.URL}}
{{range .Notes}}
"{{.Highlight}}" ([link]({{.URL}}))
{{if gt (len .Personal) 0}}> {{.Personal}}{{end}}
{{end}}
{{- end}}`

func main() {
	log.SetPrefix("ool2md: ")
	log.SetFlags(log.Lmsgprefix)
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [FLAGS] <input.csv>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if *flagTitle == "" {
		log.Fatalf("Search term is required")
	}
	if flag.NArg() < 1 {
		log.Fatalf("Missing CSV input file")
	}

	if *flagOutput == "" {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("Failed to get current working directory: %v\n", err)
		}
		*flagOutput = cwd
	} else {
		st, err := os.Stat(*flagOutput)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				if err := os.MkdirAll(*flagOutput, fs.FileMode(0770)); err != nil {
					log.Fatalf("Failed to create output directory: %v\n", err)
				}
			} else {
				log.Fatalf("Failed to verify output folder: %v\n", err)
			}
		} else if !st.IsDir() {
			log.Fatalf("Output folder %q exists and is not a directory\n", *flagOutput)
		}
	}

	tmpl, err := template.New("book-notes").Parse(BookNotesTmpl)
	if err != nil {
		log.Fatalf("Failed to parse template: %v\n", err)
	}

	inFile, err := os.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf("Failed to open CSV file: %v\n", err)
	}

	reader := csv.NewReader(inFile)
	reader.FieldsPerRecord = 9
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read CSV file: %v\n", err)
	}

	if len(records) < 1 {
		os.Exit(1)
	}

	re, err := regexp.Compile(fmt.Sprintf("(?i)%s", *flagTitle))
	if err != nil {
		log.Fatalf("Failed to compile regex: %v\n", err)
	}

	matches := make(map[string]book)
	for _, rec := range records[1:] {
		row := arr2inputRow(rec)
		if re.Match([]byte(row.title)) {
			if b, ok := matches[row.title]; ok {
				if c, ok := b.Chapters[row.chapter]; ok {
					c.Notes = append(c.Notes, row.toNote())
				} else {
					b.Chapters[row.chapter] = row.toChapter()
				}
			} else {
				matches[row.title] = row.toBook()
			}
		}
	}

	for _, m := range matches {
		if len(matches) > 1 && !*flagMultiAllow {
			if !askConfirm(fmt.Sprintf("Process book %q? ", m.Title)) {
				continue
			}
		}
		file, err := os.Create(path.Join(*flagOutput, m.Title+" - notes.md"))
		if err != nil {
			log.Fatalf("Failed to create output file: %v\n", err)
		}
		if *flagObsidian {
			m.mkObsidian()
		}
		if err := tmpl.Execute(file, m); err != nil {
			log.Fatalf("Failed to execute template: %v\n", err)
		}
	}
}

type inputRow struct {
	title        string
	authors      string
	chapter      string
	date         string
	bookURL      string
	chapterURL   string
	highlightURL string
	highlight    string
	note         string
}

func arr2inputRow(arr []string) inputRow {
	return inputRow{
		arr[0],
		arr[1],
		arr[2],
		arr[3],
		arr[4],
		arr[5],
		arr[6],
		arr[7],
		arr[8],
	}
}

func (r *inputRow) toBook() book {
	chapters := make(map[string]*chapter)
	chapters[r.chapter] = r.toChapter()
	return book{
		Title:    r.title,
		Authors:  r.authors,
		URL:      r.bookURL,
		Chapters: chapters,
	}
}

func (r *inputRow) toChapter() *chapter {
	notes := make([]*note, 0, 32)
	notes = append(notes, r.toNote())
	return &chapter{
		Title: r.chapter,
		URL:   r.chapterURL,
		Notes: notes,
	}
}

func (r *inputRow) toNote() *note {
	return &note{
		Highlight: strings.ReplaceAll(r.highlight, "\n", ""),
		URL:       r.highlightURL,
		Personal:  strings.ReplaceAll(r.note, "\n", ""),
	}
}

type book struct {
	Title    string
	Authors  string
	URL      string
	Chapters map[string]*chapter
}

type chapter struct {
	Title string
	URL   string
	Notes []*note
}

type note struct {
	Highlight string
	Personal  string
	URL       string
}

func (b *book) mkObsidian() {
	for _, c := range b.Chapters {
		for _, n := range c.Notes {
			n.Highlight = strings.ReplaceAll(n.Highlight, "#", "\\#")
		}
	}
}

func askConfirm(ask string) bool {
	ask += "(yes/[no]) "
	fmt.Print(ask)
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		r := s.Text()
		if len(r) == 0 {
			return false
		} else if r[0] == 'y' || r[0] == 'Y' {
			return true
		} else if r[0] == 'n' || r[0] == 'N' {
			return false
		}
		fmt.Printf("\n%s", ask)
	}
	return false
}
