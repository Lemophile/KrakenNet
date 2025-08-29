package config

import (
	"bufio"
	"io"
	"math/rand"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// Public, backwards-compatible surface:
//   - LoadLists(userAgentsFile, referersFile string)
//   - RandomUserAgent() string
//   - RandomReferer() string
//   - RandomMethod() string
//   - RandomPath() string
//   - RandomPathN(minLen, maxLen int) string
//   - MethodsHTTP []string

// Defaults used when lists are empty.
const (
	defaultUA      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	defaultReferer = "https://google.com/"
)

var (
	// MethodsHTTP can be overridden by callers if desired.
	MethodsHTTP = []string{"GET", "POST", "HEAD"}

	// Internals: lists are stored in atomics so reads are goroutine-safe.
	uaList  atomic.Value // []string
	refList atomic.Value // []string
)

func init() {
	// Seed global RNG once; top-level rand funcs are concurrency-safe.
	rand.Seed(time.Now().UnixNano())
	uaList.Store([]string(nil))
	refList.Store([]string(nil))
}

// LoadLists loads user agents and referers from text files.
// One entry per line. Lines starting with '#' or '//' are ignored.
// Blank lines are ignored. Pass an empty filename to skip.
func LoadLists(userAgentsFile, referersFile string) {
	if userAgentsFile != "" {
		if lst, _ := loadListFromFile(userAgentsFile); len(lst) > 0 {
			uaList.Store(lst)
		}
	}
	if referersFile != "" {
		if lst, _ := loadListFromFile(referersFile); len(lst) > 0 {
			refList.Store(lst)
		}
	}
}

// RandomUserAgent returns a random UA, or a default if none are loaded.
func RandomUserAgent() string {
	lst, _ := uaList.Load().([]string)
	return randomFromList(lst, defaultUA)
}

// RandomReferer returns a random referer, or a default if none are loaded.
func RandomReferer() string {
	lst, _ := refList.Load().([]string)
	return randomFromList(lst, defaultReferer)
}

// RandomMethod returns a random HTTP method from MethodsHTTP.
func RandomMethod() string {
	if len(MethodsHTTP) == 0 {
		return "GET"
	}
	return MethodsHTTP[rand.Intn(len(MethodsHTTP))]
}

// RandomPath returns a random URL path with length in [5,14].
func RandomPath() string { return RandomPathN(5, 14) }

// RandomPathN returns a random URL path with length in [minLen,maxLen].
// If inputs are invalid, it falls back to [5,14].
func RandomPathN(minLen, maxLen int) string {
	if minLen <= 0 || maxLen < minLen {
		minLen, maxLen = 5, 14
	}
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	n := rand.Intn(maxLen-minLen+1) + minLen
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return "/" + string(b)
}

// --- internals ---

func loadListFromFile(filename string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return loadList(f)
}

func loadList(r io.Reader) ([]string, error) {
	sc := bufio.NewScanner(r)
	var out []string
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		// treat leading '#' or '//' as comments (avoid stripping inline data)
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		out = append(out, line)
	}
	if err := sc.Err(); err != nil {
		// return whatever we parsed plus the error
		return out, err
	}
	return out, nil
}

func randomFromList(list []string, fallback string) string {
	if len(list) == 0 {
		return fallback
	}
	return list[rand.Intn(len(list))]
}
