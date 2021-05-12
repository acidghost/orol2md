# orol2md

A simple utility to convert O'Reilly Online Learning annotations to markdown.

It's currently geared to generate markdown that is importable in [Obsidian](https://obsidian.md).

## Installation

You need to have https://golang.org installed. Then issue the usual command: `go install https://github.com/acidghost/orol2md@latest`.

## Usage

First download the annotation from the learning platform (e.g. "Your O'Reilly" > "Highlights" > "Export all notes and highlights").

The basic usage allows to filter notes by book title (case-insensitive) and save it's markdown conversion to a specified folder.
The following example will, assuming there are some highlights for the book titled "Some More Title", create a new markdown file
at path `my-notes-dir/Some More Title.md`:

```sh
orol2md -o my-notes-dir -s="some.*title" exported.csv
```
