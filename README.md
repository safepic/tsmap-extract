# tsmap-extract — Recover JS/TS sources from sourcemaps for pentesters

tsmap-extract is a combined CLI for recovering original JavaScript and TypeScript source files from source maps.  
It bundles two complementary modes:

* `extract` - reconstruct sources from a local `.map` file.
* `crawl`   - crawl a web page, find JavaScript bundles, try associated `.map` files and recover sources.

The tool is written in pure Go and focuses on safe path handling (prevents path traversal), pentest-friendly features (proxy, insecure TLS for intercepting proxies), and no external runtime dependencies for the extraction logic.

------------------------------------------------------------

## Features


* `extract` subcommand: extract sources from a local `.map` file
* `crawl` subcommand: fetch a web page, discover `<script src>` entries, download `.js` and try associated `.map` files (inline base64 or external)
* Safe path anchoring with support for `..` segments while preventing files leaving the output directory
* Ignore empty `sourcesContent` when computing anchor depth
* Optional basic beautification for JS/TS (`--beautify`)
* Optional EOL normalization (`--eol unix|dos`)
* Proxy support (`--proxy`) and TLS verification skip (`--insecure`) for use with intercepting proxies (Burp/ZAP)
* Options to save downloaded `.js` and `.map` files (`--save-js`, `--save-map`)
* Concurrency control for crawling (`--concurrency`)
* Single binary with both modes; no extra runtime libraries required for extraction logic

------------------------------------------------------------

## Installation

Requires Go 1.24 or later.

### go build (recommended)
```bash
git clone git@github.com:safepic/tsmap-extract.git
cd tsmap-extract
go build -o tsmap-extract main.go
```

------------------------------------------------------------

## Usage

General form:
```
tsmap-extract <subcommand> [flags]
```

```
tsmap-extract - combined extractor and crawler

Usage:
tsmap-extract extract [flags]    Extract sources from a .map file
tsmap-extract crawl   [flags]    Crawl a page, find JS and extract .map sources

Run 'tsmap-extract <subcommand> -h' for subcommand help.
```
------------------------------------------------------------
### extract - Flags & example

Extract sources from a local `.map` file.

Flags:
* `-map <file>`          : Path to the .map file (required)
* `-out <dir>`           : Output directory (default: extracted_sources)
* `-beautify`            : Enable basic beautification of JS/TS output
* `-eol unix|dos`        : Normalize line endings to LF (unix) or CRLF (dos)

Example:

```bash
tsmap-extract extract -map dist/app.js.map -out ./sources --beautify --eol unix
```

Example output:
```bash
Written: sources/src/app.ts
Written: sources/src/utils/math.ts
Skipped (no content): ../node_modules/core-js/internals/object-keys.js

Summary: 2 written, 1 skipped
```
------------------------------------------------------------

------------------------------------------------------------
### crawl - Flags & example

Crawl a page, fetch JS bundles, try to find or derive `.map` URLs and extract sources.

Flags:
* `-url <url>`           : Root page URL to crawl (required)
* `-out <dir>`           : Output base directory (default: recovered)
* `-beautify`            : Enable basic beautification of JS/TS output
* `-eol unix|dos`        : Normalize line endings to LF or CRLF
* `-concurrency <n>`     : Parallel downloads (default: 4)
* `-user-agent <str>`    : User-Agent header (default: tsmap-crawl/1.0)
* `--save-js`            : Save downloaded .js files beside recovered sources
* `--save-map`           : Save downloaded .map files beside recovered sources
* `--proxy <url>`        : Proxy (e.g. http://127.0.0.1:8080)
* `--insecure`           : Disable TLS verification (useful with intercepting proxies)


```bash
tsmap-extract crawl -url https://example.com/ -out ./sources --beautify --eol unix 
```




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

- Files cannot escape the target output directory (anti-traversal).
- Leading `..` in sourcemap paths are handled by an internal anchor, but resulting files remain inside `-out`.
- Empty `sourcesContent` entries are ignored when computing anchor depth (avoids deep unused anchors).
- No network access is performed by `extract` (local only).
- `crawl` performs network requests; respect target site rules and legal constraints when pentesting.

------------------------------------------------------------

## Recommendations for pentesters

- Use `--proxy` + `--insecure` with Burp or ZAP to inspect HTTP/HTTPS traffic.
- If possible, import the Burp CA to avoid using `--insecure`.
- Use `--save-js` and `--save-map` to keep original artifacts for later analysis.
- Use `--concurrency` to tune speed vs. politeness depending on the target.


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

