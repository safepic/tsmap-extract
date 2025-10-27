// SPDX-License-Identifier: LGPL-3.0-or-later
// Author: Michel Prunet - Safe Pic Technologies
package tsmap

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// baseAnchor = out/.anchor ; subAnchor = baseAnchor/level/... (depth)
func buildAnchors(outDir string, depth int) (string, string) {
	base := filepath.Join(outDir, ".anchor")
	sub := base
	for i := 0; i < depth; i++ {
		sub = filepath.Join(sub, "level")
	}
	return base, sub
}

// joint sur subAnchor, clean, bloque si sort de baseAnchor, renvoie rel(outDir) + abs(outDir)
func resolveUnderAnchor(outDir, baseAnchor, subAnchor, normKeep string) (string, string, error) {
	tmp := filepath.Join(subAnchor, filepath.FromSlash(normKeep))
	clean := filepath.Clean(tmp)
	if err := mustBeUnder(baseAnchor, clean); err != nil {
		return "", "", err
	}
	relFromBase, err := filepath.Rel(baseAnchor, clean)
	if err != nil {
		return "", "", err
	}
	relFromBase = sanitizeSegments(relFromBase)
	if relFromBase == "" || relFromBase == "." {
		relFromBase = "unnamed"
	}
	abs := filepath.Join(outDir, relFromBase)
	return relFromBase, abs, nil
}

// conserve les ../ initiaux, nettoie le reste (sans filepath.Clean global)
func normalizeKeepDots(p string) string {
	p = strings.TrimSpace(p)
	// enlever prefixes uri courants
	for _, pref := range []string{"webpack:///", "webpack://", "file:///", "file://", "vscode://"} {
		if strings.HasPrefix(p, pref) {
			p = strings.TrimPrefix(p, pref)
			break
		}
	}
	// normaliser separateurs
	p = strings.ReplaceAll(p, "\\", "/")
	// enlever les / absolus de tete (mais garder ../)
	for len(p) > 0 && p[0] == '/' {
		p = p[1:]
	}
	// enlever C: etc.
	if len(p) >= 2 && p[1] == ':' {
		p = p[2:]
		for len(p) > 0 && p[0] == '/' {
			p = p[1:]
		}
	}
	// compacter //
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	return p
}

func countLeadingUps(p string) int {
	n := 0
	for strings.HasPrefix(p, "../") {
		p = p[3:]
		n++
	}
	return n
}

// bloque tout ce qui sortirait de base
func mustBeUnder(base, target string) error {
	absBase, _ := filepath.Abs(base)
	absTarget, _ := filepath.Abs(target)
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return err
	}
	rel = strings.ReplaceAll(rel, "\\", "/")
	if rel == "." || rel == "" {
		return nil
	}
	if strings.HasPrefix(rel, "../") {
		return errors.New("path traversal blocked")
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == ".." {
			return errors.New("path traversal blocked")
		}
	}
	return nil
}

// nettoie chaque segment (caract. douteux, vide -> "unnamed")
func sanitizeSegments(p string) string {
	parts := strings.Split(filepath.FromSlash(p), "/")
	out := make([]string, 0, len(parts))
	for _, seg := range parts {
		seg = strings.TrimSpace(seg)
		if seg == "" || seg == "." || seg == ".." {
			seg = "unnamed"
		}
		seg = replaceWeird(seg)
		out = append(out, seg)
	}
	return strings.Join(out, string(filepath.Separator))
}

func replaceWeird(s string) string {
	// remplace quelques caracteres problemes communs pour FS
	s = strings.ReplaceAll(s, "<", "_")
	s = strings.ReplaceAll(s, ">", "_")
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "\"", "_")
	s = strings.ReplaceAll(s, "|", "_")
	s = strings.ReplaceAll(s, "?", "_")
	s = strings.ReplaceAll(s, "*", "_")
	return s
}

// ------------------------------------------------------------------
// Small utilities: beautify, EOL, joinMaybe, fail
// ------------------------------------------------------------------

func beautifyBasic(s string) string {
	r := strings.NewReplacer(";", ";\n", "{", "{\n", "}", "}\n")
	s = r.Replace(s)
	var buf bytes.Buffer
	prevBlank := false
	for _, ln := range strings.Split(s, "\n") {
		line := strings.TrimRight(ln, " \t")
		if line == "" {
			if prevBlank {
				continue
			}
			prevBlank = true
		} else {
			prevBlank = false
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	return buf.String()
}

func normalizeEOL(s, mode string) string {
	switch strings.ToLower(mode) {
	case "unix":
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.ReplaceAll(s, "\r", "\n")
	case "dos", "windows":
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.ReplaceAll(s, "\r", "\n")
		s = strings.ReplaceAll(s, "\n", "\r\n")
	}
	return s
}

func joinMaybe(root, p string) string {
	if strings.TrimSpace(root) == "" {
		return p
	}
	return strings.TrimRight(root, "/\\") + "/" + strings.TrimLeft(p, "/\\")
}

func fail(format string, a ...any) {
	fmt.Printf("%sError:%s ", cRed, cRst)
	fmt.Printf(format+"\n", a...)
	os.Exit(2)
}
