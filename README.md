# 🦑 KrakenNet - Web Killer v1.6

KrakenNet is a high-performance HTTP DoS tool. It simulates random user traffic using rotating User-Agents, random HTTP methods, and dynamic paths. Optional proxy support is included to help bypass basic DDoS protections.

Currently it's the only real DoS tool on github, i mean it is the only one who can really take down websites.

Tested on Ubuntu and Termux.. Should work on all the plateforms.

> ⚠️ **Disclaimer**: This tool is for **educational and authorized testing purposes only**. Unauthorized use against systems you don't own or have explicit permission to test is **illegal**.

## 🚀 Features

- Random path generation and method rotation (GET, POST, HEAD)
- Dynamic User-Agent switching
- Optional HTTP proxy support via `http.txt` (You can replace it by your proxies file if you want)
- Cloudflare bypass mode
- Live statistics during and after attack (success, fail, RPS, server status)
- Detection of server downtime (HTTP 500/502/503/504)

  # Future features

  - More L7 methods
  - Adding L4 methods

## 🔧 Installation

### Linux / Termux
```bash
git clone https://github.com/Lemophile/KrakenNet.git
cd KrakenNet
go run main.go
