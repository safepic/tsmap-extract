// SPDX-License-Identifier: LGPL-3.0-or-later
// Author: Michel Prunet - Safe Pic Technologies
package tsmap

type sourceMap struct {
	Version        int      `json:"version"`
	File           string   `json:"file"`
	Sources        []string `json:"sources"`
	SourcesContent []string `json:"sourcesContent"`
	SourceRoot     string   `json:"sourceRoot"`
}
