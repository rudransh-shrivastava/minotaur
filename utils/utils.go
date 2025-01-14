package utils

import (
	"fmt"
	"time"

	"github.com/rudransh-shrivastava/minotaur/proxy"
)

func LogLoop(servers *[]proxy.Server) {
	for {
		time.Sleep(10 * time.Second)
		fmt.Println("++++++++++++++++++++")
		fmt.Println("Server status")
		for _, server := range *servers {
			fmt.Printf("server: %s, count: %d\n", server.URL, server.Count)
		}
	}
}
