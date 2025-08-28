package pkg

import (
	"fmt"
	color "github.com/vexsx/KrakenNet/config"
)

func PrintBanner() {
	banner := `

    __ __ ____  ___    __ __ _______   __   _   ______________
   / //_// __ \/   |  / //_// ____/ | / /  / | / / ____/_  __/
  / ,<  / /_/ / /| | / ,<  / __/ /  |/ /  /  |/ / __/   / /   
 / /| |/ _, _/ ___ |/ /| |/ /___/ /|  /  / /|  / /___  / /    
/_/ |_/_/ |_/_/  |_/_/ |_/_____/_/ |_/  /_/ |_/_____/ /_/     
                                                              
`

	authors := `
	KRAKEN NET - v2.2
	Made by Piwiii2.0 Edited by Vexsx

`

	fmt.Print(color.Cyan)
	fmt.Println(banner)
	fmt.Print(color.Reset)
	fmt.Print(color.Red)
	fmt.Println(authors)
	fmt.Print(color.Reset)

}
