package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/rudransh-shrivastava/minotaur/proxy"
)

func LogLoop(ctx context.Context, servers *[]proxy.Server) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fmt.Println("++++++++++++++++++++")
			fmt.Println("Server status")
			for i := range *servers {
				fmt.Printf("server: %s, count: %d\n", (*servers)[i].URL, (*servers)[i].Count)
			}
		}
	}
}
