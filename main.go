package main

import (
	"bufio"
	"fmt"
	"github.com/vexsx/KrakenNet/config"
	"github.com/vexsx/KrakenNet/pkg"
	"math/rand"
	"os"
	"strings"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	pkg.PrintBanner()
	reader := bufio.NewReader(os.Stdin)

	for {
		pkg.RunAttack()

		fmt.Print(color.Yellow + "\n🔄 Do you want to start another attack? (y/n): " + color.Reset)
		again, _ := reader.ReadString('\n')
		again = strings.TrimSpace(strings.ToLower(again))

		if again != "y" {
			fmt.Println(color.Green + "👋 Bye! Hope you liked your attacks" + color.Reset)
			break
		}
	}
}
