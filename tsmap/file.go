// SPDX-License-Identifier: LGPL-3.0-or-later
// Author: Michel Prunet - Safe Pic Technologies
package tsmap

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func RunExtract(args []string) {
	fs := flag.NewFlagSet("tsmap-extract extract", flag.ExitOnError)
	mapPath := fs.String("map", "", "Path to .map file")
	outDir := fs.String("out", "extracted_sources", "Output directory")
	beautify := fs.Bool("beautify", false, "Beautify minimal JS/TS")
	eol := fs.String("eol", "", "Line endings: unix|dos")
	fs.Parse(args)

	if strings.TrimSpace(*mapPath) == "" {
		fs.Usage()
	}

	raw, err := os.ReadFile(*mapPath)
	if err != nil {
		fail("Read .map: %v", err)
	}
	var sm sourceMap
	if err := json.Unmarshal(raw, &sm); err != nil {
		fail("Invalid sourcemap JSON: %v", err)
	}
	if len(sm.Sources) == 0 {
		fail("No 'sources' in sourcemap")
	}
	_ = os.MkdirAll(*outDir, 0755)

	// Calcul ancrage
	maxUp := computeMaxLeadingUps(sm)
	baseAnchor, subAnchor := buildAnchors(*outDir, maxUp)

	written, skipped := 0, 0

	for i, s := range sm.Sources {
		content := ""
		if i < len(sm.SourcesContent) {
			content = sm.SourcesContent[i]
		}
		if strings.TrimSpace(content) == "" {
			fmt.Printf("%sSkipped%s (no content): %s\n", cYel, cRst, s)
			skipped++
			continue
		}

		// Normaliser en conservant les ../
		norm := normalizeKeepDots(joinMaybe(sm.SourceRoot, s))

		// Résoudre via ancrage
		rel, abs, err := resolveUnderAnchor(*outDir, baseAnchor, subAnchor, norm)
		if err != nil {
			fmt.Printf("%sSkipped%s (path blocked): %s\n", cYel, cRst, s)
			skipped++
			continue
		}

		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			fail("Create dir: %v", err)
		}

		if *beautify {
			content = beautifyBasic(content)
		}
		content = normalizeEOL(content, *eol)

		if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
			fail("Write file: %v", err)
		}
		fmt.Printf("%sWritten%s: %s\n", cGrn, cRst, filepath.Join(*outDir, rel))
		written++
	}

	fmt.Printf("\n%sSummary%s: %d written, %d skipped\n", cCyn, cRst, written, skipped)
}

// ---------- Anchoring & path logic ----------

// Calcule le nombre max de "../" en ignorant les fichiers vides
func computeMaxLeadingUps(sm sourceMap) int {
	maxUp := 0
	for i, s := range sm.Sources {
		if i < len(sm.SourcesContent) {
			if strings.TrimSpace(sm.SourcesContent[i]) == "" {
				continue // on ignore les fichiers sans contenu
			}
		}
		p := normalizeKeepDots(joinMaybe(sm.SourceRoot, s))
		if n := countLeadingUps(p); n > maxUp {
			maxUp = n
		}
	}
	return maxUp
}
