package main

import (
	"fmt"
	"os"

	"tsmap-extract.safepic.fr/tsmap"
)

func usage() {
	fmt.Println("tsmap-extract - combined extractor and crawler")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  tsmap-extract extract [flags]    Extract sources from a .map file")
	fmt.Println("  tsmap-extract crawl   [flags]    Crawl a page, find JS and extract .map sources")
	fmt.Println()
	fmt.Println("Run 'tsmap-extract <subcommand> -h' for subcommand help.")
}

// ---------- main ----------
func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]

	switch cmd {
	case "extract":
		tsmap.RunExtract(os.Args[2:])
	case "crawl":
		tsmap.RunCrawl(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", cmd)
		usage()
		os.Exit(2)
	}
}
