# tsmap-extract — Recover JS/TS sources from sourcemaps for pentesters

tsmap-extract is a lightweight, pure-Go tool to recover original JavaScript and TypeScript source files from source map (.map) files. It is designed for security professionals and pentesters who need to extract readable code from minified or bundled assets during assessments, incident response, or forensic analysis. Features include safe path resolution (prevents path traversal), support for ../ segments, optional basic beautification, and EOL normalization.

It is written in pure Go, with no external dependencies, and includes safe path handling to prevent path traversal and ensure correct reconstruction of directory structures.

------------------------------------------------------------

## Features

* Reads standard JS/TS sourcemap files (.js.map, .ts.map, etc.)
* Recreates the original source files in a specified output directory
* Prevents directory traversal (files cannot escape the output directory)
* Supports relative paths with .. segments correctly and safely
* Ignores empty source entries when determining directory depth
* Optional basic beautification for JS/TS files (--beautify)
* Optional end-of-line normalization (--eol unix or --eol dos)
* Pure Go standard library (no external packages)
* Works on Linux, macOS, Windows

------------------------------------------------------------

## Installation

Requires Go 1.21 or later.

git clone https://github.com/yourusername/tsmap-extract.git
cd tsmap-extract
go build -o tsmap-extract tsmap-extract.go

------------------------------------------------------------

## Usage

tsmap-extract -map <file.map> [options]

Options:

- -map : Path to the .map file (required)
- -out : Output directory (default: extracted_sources)
- --beautify : Enables basic readable formatting for JS/TS files
- --eol unix or --eol dos : Normalize line endings (LF or CRLF)

------------------------------------------------------------

## Example

tsmap-extract -map dist/app.js.map -out ./sources --beautify --eol unix

Example output:

Written: sources/src/app.ts
Written: sources/src/utils/math.ts
Skipped (no content): ../node_modules/core-js/internals/object-keys.js

Summary: 2 written, 1 skipped

------------------------------------------------------------

## How path handling works

Some sourcemaps contain paths with segments like:

../../src/utils.js

To handle this safely:

1. The tool computes the maximum number of leading .. segments across all non-empty sources.
2. It creates an internal anchor directory tree inside the output folder.
3. Paths are resolved relative to that anchor.
4. Resulting files are always inside the output directory.

Example:

Input:
../../src/foo.js

Output:
extracted_sources/src/foo.js

------------------------------------------------------------

## Security

- Output files always remain inside the output directory
- Safe resolution of all relative paths
- Sanitization of file names
- No network access
- No execution of extracted content
- Empty or missing sources are ignored

------------------------------------------------------------

## Cross compilation (optional)

A Makefile can be used to build binaries for multiple platforms:

make linux
make windows
make darwin-arm64

Outputs are placed in the bin/ directory.

------------------------------------------------------------

## License

LGPL-3.0-or-later

See the LICENSE file for details.

------------------------------------------------------------

## Author

Michel Prunet - Safe Pic Technologies

------------------------------------------------------------

