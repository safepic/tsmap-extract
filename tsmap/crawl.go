// SPDX-License-Identifier: LGPL-3.0-or-later
// Author: Michel Prunet - Safe Pic Technologies
package tsmap

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

var client = &http.Client{
	Timeout: 25 * time.Second,
}

var reSourceMapInline = regexp.MustCompile(`(?m)//[#@]\s*sourceMappingURL=data:application/json(?:;charset=[^;]+)?;base64,([A-Za-z0-9+/=]+)`)
var reSourceMapComment = regexp.MustCompile(`(?m)//[#@]\s*sourceMappingURL\s*=\s*(.+)$`)

func RunCrawl(args []string) {
	fs := flag.NewFlagSet("tsmap-extract crawl", flag.ExitOnError)
	urlRoot := fs.String("url", "", "Root page URL to crawl (required)")
	outDir := fs.String("out", "recovered", "Output base directory")
	beautify := fs.Bool("beautify", false, "Beautify minimal JS/TS")
	eol := fs.String("eol", "", "Normalize EOL: unix|dos")
	concurrency := fs.Int("concurrency", 4, "Parallel downloads")
	userAgent := fs.String("user-agent", "tsmap-crawl/1.0", "User-Agent header")
	saveJS := fs.Bool("save-js", false, "Save downloaded .js files alongside recovered sources")
	saveMap := fs.Bool("save-map", false, "Save downloaded .map files alongside recovered sources")
	proxy := fs.String("proxy", "", "Proxy URL (e.g. http://127.0.0.1:8080)")
	insecure := fs.Bool("insecure", false, "Skip TLS verification, usefull with burpsuite")

	fs.Parse(args)
	transport := &http.Transport{}
	if *proxy != "" {
		proxyURL, err := url.Parse(*proxy)
		if err != nil {
			fail("Invalid proxy URL: %v", err)
		}

		transport.Proxy = http.ProxyURL(proxyURL)
		transport.ForceAttemptHTTP2 = false
		transport.TLSHandshakeTimeout = 30 * time.Second
		fmt.Printf("%sUsing proxy:%s %s\n", cCyn, cRst, proxyURL.String())
	} else {
		transport.Proxy = http.ProxyFromEnvironment
	}

	// Option to skip TLS verification (for Burp/ZAP interception)
	if *insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		fmt.Printf("%sWarning:%s TLS verification disabled (insecure mode)\n", cYel, cRst)
	}
	// override client with proxy-enabled transport
	client = &http.Client{
		Timeout:   25 * time.Second,
		Transport: transport,
	}
	if strings.TrimSpace(*urlRoot) == "" {
		fmt.Fprintln(os.Stderr, "Missing -url")
		flag.Usage()
		os.Exit(2)
	}

	rootURL, err := url.Parse(*urlRoot)
	if err != nil {
		fail("Invalid url: %v", err)
	}

	// fetch root
	fmt.Printf("Fetching: %s\n", rootURL.String())
	req, _ := http.NewRequestWithContext(context.Background(), "GET", rootURL.String(), nil)
	req.Header.Set("User-Agent", *userAgent)
	resp, err := client.Do(req)
	if err != nil {
		fail("Failed to fetch root URL: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		fail("HTTP error fetching root: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fail("Read body: %v", err)
	}

	// parse HTML scripts with x/net/html
	scripts := parseScriptsHTML(string(body), rootURL)
	if len(scripts) == 0 {
		fmt.Println("No external script src found on page.")
	}

	// worker pool
	sem := make(chan struct{}, *concurrency)
	var wg sync.WaitGroup
	results := make(chan string, len(scripts))
	endWrite := make(chan struct{})
	writtenTotal := 0
	go func() {
		for r := range results {
			fmt.Println(r)
			if strings.HasPrefix(r, "WRITTEN:") {
				writtenTotal++
			}
		}
		endWrite <- struct{}{}
	}()

	for _, s := range scripts {
		wg.Add(1)
		go func(scriptURL *url.URL) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			processScript(scriptURL, rootURL, *outDir, *beautify, *eol, *userAgent, *saveJS, *saveMap, results)
		}(s)
	}

	wg.Wait()
	close(results)
	<-endWrite
	fmt.Printf("\nDone. Scripts processed: %d. Sources written groups: %d\n", len(scripts), writtenTotal)
}

// parseScriptsHTML uses golang.org/x/net/html to find <script src=...>
func parseScriptsHTML(src string, base *url.URL) []*url.URL {
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		// fallback to simple regex if parse fails
		return parseScriptsRegex(src, base)
	}
	var out []*url.URL
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "script") {
			for _, a := range n.Attr {
				if strings.EqualFold(a.Key, "src") && strings.TrimSpace(a.Val) != "" {
					raw := strings.TrimSpace(a.Val)
					u, err := url.Parse(raw)
					if err == nil {
						out = append(out, base.ResolveReference(u))
					}
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	// dedupe
	seen := make(map[string]bool)
	var dedup []*url.URL
	for _, u := range out {
		if u == nil {
			continue
		}
		if !seen[u.String()] {
			seen[u.String()] = true
			dedup = append(dedup, u)
		}
	}
	return dedup
}

// fallback regex parser
func parseScriptsRegex(htmlSrc string, base *url.URL) []*url.URL {
	re := regexp.MustCompile(`(?i)<script[^>]+src\s*=\s*['"]([^'"]+)['"]`)
	matches := re.FindAllStringSubmatch(htmlSrc, -1)
	var out []*url.URL
	for _, m := range matches {
		raw := m[1]
		u, err := url.Parse(raw)
		if err == nil {
			out = append(out, base.ResolveReference(u))
		}
	}
	seen := make(map[string]bool)
	var dedup []*url.URL
	for _, u := range out {
		if u == nil {
			continue
		}
		if !seen[u.String()] {
			seen[u.String()] = true
			dedup = append(dedup, u)
		}
	}
	return dedup
}

func processScript(scriptURL *url.URL, rootURL *url.URL, outBase string, beautify bool, eol string, userAgent string, saveJS, saveMap bool, results chan<- string) {
	results <- fmt.Sprintf("Processing: %s", scriptURL.String())

	// fetch .js
	jsBytes, err := fetchURLBytes(scriptURL.String(), userAgent)
	if err != nil {
		results <- fmt.Sprintf("%sFailed to fetch script: %v%s", cYel, err, cRst)
		return
	}
	jsText := string(jsBytes)

	// optional save js
	if saveJS {
		hostPath := hostPathForURL(rootURL, scriptURL)
		outDir := filepath.Join(outBase, hostPath)
		_ = os.MkdirAll(outDir, 0755)
		jsName := filepath.Base(scriptURL.Path)
		if jsName == "" {
			jsName = "script.js"
		}
		_ = os.WriteFile(filepath.Join(outDir, jsName), jsBytes, 0644)
	}

	// 1) inline base64 map
	if m := reSourceMapInline.FindStringSubmatch(jsText); len(m) > 1 {
		b64 := m[1]
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			results <- fmt.Sprintf("%sInline map decode error: %v%s", cYel, err, cRst)
		} else {
			hostPath := hostPathForURL(rootURL, scriptURL)
			nwritten, err := processMapBytes(data, outBase, hostPath, beautify, eol, saveMap, "")
			if err != nil {
				results <- fmt.Sprintf("%sError processing inline map: %v%s", cYel, err, cRst)
			} else {
				results <- fmt.Sprintf("WRITTEN:%d inline map for %s", nwritten, scriptURL.String())
			}
			return
		}
	}

	// 2) sourceMappingURL comment
	if m := reSourceMapComment.FindStringSubmatch(jsText); len(m) > 1 {
		ref := strings.TrimSpace(m[1])
		ref = strings.Trim(ref, "\"'")
		// Map ref can be relative; resolve against scriptURL
		mapURL, err := scriptURL.Parse(ref)
		if err == nil {
			data, err := fetchURLBytes(mapURL.String(), userAgent)
			if err != nil {
				results <- fmt.Sprintf("%sFailed to fetch map %s: %v%s", cYel, mapURL.String(), err, cRst)
			} else {
				hostPath := hostPathForURL(rootURL, scriptURL)
				nwritten, err := processMapBytes(data, outBase, hostPath, beautify, eol, saveMap, mapURL.String())
				if err != nil {
					results <- fmt.Sprintf("%sError processing map %s: %v%s", cYel, mapURL.String(), err, cRst)
				} else {
					results <- fmt.Sprintf("WRITTEN:%d map for %s", nwritten, mapURL.String())
				}
				return
			}
		}
	}

	// 3) try script.js.map
	tryMapURL := scriptURL.ResolveReference(&url.URL{Path: scriptURL.Path + ".map"})
	data, err := fetchURLBytes(tryMapURL.String(), userAgent)
	if err == nil {
		hostPath := hostPathForURL(rootURL, scriptURL)
		nwritten, err := processMapBytes(data, outBase, hostPath, beautify, eol, saveMap, tryMapURL.String())
		if err != nil {
			results <- fmt.Sprintf("%sError processing map %s: %v%s", cYel, tryMapURL.String(), err, cRst)
		} else {
			results <- fmt.Sprintf("WRITTEN:%d map for %s", nwritten, tryMapURL.String())
		}
		return
	}

	results <- fmt.Sprintf("%sNo sourcemap for %s%s", cYel, scriptURL.String(), cRst)
}

func fetchURLBytes(u string, userAgent string) ([]byte, error) {

	req, _ := http.NewRequestWithContext(context.Background(), "GET", u, nil)
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func hostPathForURL(rootURL, scriptURL *url.URL) string {
	host := scriptURL.Hostname()
	dir := filepath.Dir(scriptURL.Path)
	if dir == "." || dir == "/" {
		dir = ""
	} else {
		dir = strings.Trim(dir, "/")
	}
	if dir == "" {
		return host
	}
	return filepath.Join(host, dir)
}

func processMapBytes(mapData []byte, outBase, hostPath string, beautify bool, eol string, saveMap bool, mapURL string) (int, error) {
	var sm sourceMap
	if err := json.Unmarshal(mapData, &sm); err != nil {
		return 0, err
	}
	outRoot := filepath.Join(outBase, hostPath)
	_ = os.MkdirAll(outRoot, 0755)

	// optional: save map file
	if saveMap {
		mapName := "sourcemap.json"
		if mapURL != "" {
			mapName = filepath.Base(mapURL)
			if mapName == "" {
				mapName = "sourcemap.json"
			}
		}
		_ = os.WriteFile(filepath.Join(outRoot, mapName), mapData, 0644)
	}

	maxUp := computeMaxLeadingUpsFiltered(sm)
	baseAnchor, subAnchor := buildAnchors(outRoot, maxUp)

	written := 0
	for i, src := range sm.Sources {
		content := ""
		if i < len(sm.SourcesContent) {
			content = sm.SourcesContent[i]
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		norm := normalizeKeepDots(joinMaybe(sm.SourceRoot, src))
		_, abs, err := resolveUnderAnchor(outRoot, baseAnchor, subAnchor, norm)
		if err != nil {
			// skip problematic path
			continue
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			return written, err
		}
		if beautify {
			content = beautifyBasic(content)
		}
		content = normalizeEOL(content, eol)
		if err := os.WriteFile(abs, []byte(content), 0644); err != nil {
			return written, err
		}
		written++
	}
	return written, nil
}

// ------------------------------------------------------------------
// Path / anchor helpers (same logic as earlier safe version)
// ------------------------------------------------------------------

func computeMaxLeadingUpsFiltered(sm sourceMap) int {
	maxUp := 0
	for i, s := range sm.Sources {
		if i < len(sm.SourcesContent) {
			if strings.TrimSpace(sm.SourcesContent[i]) == "" {
				continue
			}
		}
		p := normalizeKeepDots(joinMaybe(sm.SourceRoot, s))
		if n := countLeadingUps(p); n > maxUp {
			maxUp = n
		}
	}
	return maxUp
}
