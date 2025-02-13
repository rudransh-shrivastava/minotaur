package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/olekukonko/tablewriter"
)

type BenchmarkResult struct {
	URL           string
	Latency       string
	ReqPerSec     string
	TransferSec   string
	TotalRequests string
	Errors        string
}

func runWrkCommand(command []string) (BenchmarkResult, error) {
	var result BenchmarkResult

	cmd := exec.Command(command[0], command[1:]...)

	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		return result, fmt.Errorf("error running command: %v\nOutput: %s", err, out.String())
	}

	// Parse the output
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Running") {
			result.URL = strings.Split(line, "@")[1]
		} else if strings.Contains(line, "Latency") {
			fields := strings.Fields(line)
			result.Latency = fields[1]
		} else if strings.Contains(line, "Requests/sec") {
			fields := strings.Fields(line)
			result.ReqPerSec = fields[1]
		} else if strings.Contains(line, "Transfer/sec") {
			fields := strings.Fields(line)
			result.TransferSec = fields[1]
		} else if strings.Contains(line, "Socket errors") {
			result.Errors = line
		} else if strings.Contains(line, "requests in") {
			fields := strings.Fields(line)
			result.TotalRequests = fields[0]
		}
	}

	return result, nil
}

func newCommands(port int) [][]string {
	return [][]string{
		// Basic endpoint that returns "foo"
		// Cacheable
		{"wrk", "-t6", "-c400", "-d30s", "-H", "Connection: keep-alive", fmt.Sprintf("http://localhost:%d/foo", port)},

		// Do not cache this endpoints
		// /random-delay responds after a random delay (0-200ms)
		{"wrk", "-t6", "-c400", "-d30s", "-H", "Connection: keep-alive", "-H", "Cache-Control: no-cache", fmt.Sprintf("http://localhost:%d/random-delay", port)},

		// Cacheable endpoint
		{"wrk", "-t6", "-c400", "-d30s", "-H", "Connection: keep-alive", fmt.Sprintf("http://localhost:%d/cached/item-99", port)},

		// Dynamic content
		{"wrk", "-t6", "-c400", "-d30s", "-H", "Connection: keep-alive", "-H", "Cache-Control: no-cache", fmt.Sprintf("http://localhost:%d/dynamic", port)},
	}
}

func main() {
	fmt.Println("Running wrk benchmarks on nginx...")
	nginxCommands := newCommands(8080)

	nginxResults := []BenchmarkResult{}

	for _, cmd := range nginxCommands {
		fmt.Printf("Running: %s\n", cmd)
		result, err := runWrkCommand(cmd)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		nginxResults = append(nginxResults, result)
	}

	fmt.Println("Running wrk benchmakrs on minotaur...")

	minotaurCommands := newCommands(59000)
	minotaurResults := []BenchmarkResult{}

	for _, cmd := range minotaurCommands {
		fmt.Printf("Running: %s\n", cmd)
		result, err := runWrkCommand(cmd)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		minotaurResults = append(minotaurResults, result)
	}

	fmt.Println("Results: ")
	table := tablewriter.NewWriter(os.Stdout)

	table.SetHeader([]string{"Proxy -> Endpoint", "Latency", "Req/Sec", "Transfer/Sec", "Total Requests", "Errors"})

	for i := range len(nginxResults) {
		// URL: http://localhost:8080/foo
		endpoint := strings.Split(nginxResults[i].URL, "8080")[1]
		table.Append([]string{
			fmt.Sprintf("Nginx -> %s", endpoint),
			nginxResults[i].Latency,
			nginxResults[i].ReqPerSec,
			nginxResults[i].TransferSec,
			nginxResults[i].TotalRequests,
			nginxResults[i].Errors,
		})
		table.Append([]string{
			fmt.Sprintf("Minotaur -> %s", endpoint),
			minotaurResults[i].Latency,
			minotaurResults[i].ReqPerSec,
			minotaurResults[i].TransferSec,
			minotaurResults[i].TotalRequests,
			minotaurResults[i].Errors,
		})
	}
	table.SetRowLine(true)
	table.SetBorders(tablewriter.Border{Left: true, Top: true, Right: true, Bottom: true})
	table.SetCenterSeparator("+")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.Render()
}
