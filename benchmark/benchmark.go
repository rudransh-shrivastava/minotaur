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

func runBenchmark(port int) []BenchmarkResult {
	commands := newCommands(port)
	results := []BenchmarkResult{}

	for _, cmd := range commands {
		fmt.Printf("Running: %s\n", cmd)
		result, err := runWrkCommand(cmd)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		results = append(results, result)
	}
	return results
}

func main() {
	var mode string
	if len(os.Args) < 2 {
		fmt.Println("Usage: benchmark <mode>")
		fmt.Println("<mode> can be 'nginx' 'minotaur' or 'both'.")
	} else {
		mode = os.Args[1]
	}
	if mode != "nginx" && mode != "minotaur" && mode != "both" {
		fmt.Println("Invalid mode. Please use 'nginx', 'minotaur' or 'both'.")
		os.Exit(1)
	}

	nginxResults := make([]BenchmarkResult, 0)
	minotaurResults := make([]BenchmarkResult, 0)

	if mode == "both" {
		fmt.Println("Running wrk benchmarks on nginx...")
		nginxResults = runBenchmark(8080)
		fmt.Println("Running wrk benchmakrs on minotaur...")
		minotaurResults = runBenchmark(59000)
	}
	if mode == "nginx" {
		fmt.Println("Running wrk benchmarks on nginx...")
		nginxResults = runBenchmark(8080)
	}
	if mode == "minotaur" {
		fmt.Println("Running wrk benchmakrs on minotaur...")
		minotaurResults = runBenchmark(59000)
	}

	fmt.Println("Results: ")
	table := tablewriter.NewWriter(os.Stdout)

	table.SetHeader([]string{"Proxy -> Endpoint", "Latency", "Req/Sec", "Transfer/Sec", "Total Requests", "Errors"})

	if mode == "both" {
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
	}
	if mode == "nginx" {
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
		}
	}
	if mode == "minotaur" {
		for i := range len(minotaurResults) {
			// URL: http://localhost:8080/foo
			endpoint := strings.Split(minotaurResults[i].URL, "59000")[1]
			table.Append([]string{
				fmt.Sprintf("Minotaur -> %s", endpoint),
				minotaurResults[i].Latency,
				minotaurResults[i].ReqPerSec,
				minotaurResults[i].TransferSec,
				minotaurResults[i].TotalRequests,
				minotaurResults[i].Errors,
			})
		}
	}

	// Other table configurations
	table.SetRowLine(true)
	table.SetBorders(tablewriter.Border{Left: true, Top: true, Right: true, Bottom: true})
	table.SetCenterSeparator("+")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.Render()
}
