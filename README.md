# Minotaur

Minotaur is a high performance reverse proxy that supports load balancing, SSL/TLS encryption, and caching.

## Table of Contents

- [Introduction](#introduction)
- [Benchmarks](#benchmarks-minotaur-vs-nginx)
- [Other Features](#other-features)
- [How Minotaur Works](#how-minotaur-works)
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
- [Usage](#usage)
- [License](#license)

## Introduction

Minotaur is a high-performance reverse proxy server that acts as an intermediary between clients and backend servers. It is designed to handle a large number of concurrent connections and distribute incoming requests across multiple backend servers.

One of the main advantages of Minotaur is its ability to distribute requests using an algorithm that takes into account the performance of each server and adjusts the distribution of requests accordingly. This ensures that faster servers handle more requests, leading to better performance and reduced latency.

More on this algorithm in [How Minotaur Works](#how-minotaur-works)

## Benchmarks (Minotaur vs NGINX)
NGINX is a popular reverse proxy server known for its high performance and low resource usage.

In this benchmark, we compare the performance of Minotaur with NGINX to see how they handle a large number of concurrent requests in various senerios.

*NOTE*: All benchmarks were conducted on a machine with the following specifications:
- **CPU**: 12th Gen Intel i3-1215U (8) @ 4.400GHz
- **RAM**: 16GB
- **OS**: EndeavourOS (Arch Linux)

Nginx has a few load balancing algorithms that can be tuned to improve performance. For this benchmark, I used all its different algorithms against Minotaurs singular algorithm.
Read more about their algorithms here: [Nginx Docs](https://docs.nginx.com/nginx/admin-guide/load-balancer/http-load-balancer/#choosing-a-load-balancing-method)

NGINX Load Balancing Algorithms used for this benchmark:
- Weighted Round Robin (Default)
- Weighted Least Connections
- IP Hash
- Random

*Note*: `Least Time` algorithm was not used as its only available for NGINX Plus.

### Handlers and Servers
I used 5 servers to handle requests, each server was a simple HTTP server that responded with a simple JSON response. The servers were running on different ports on the same machine.
There were artificial delays set on each server to simulate different response times.

- Server 1: 50ms delay
- Server 2: 100ms delay
- Server 3: 150ms delay
- Server 4: 200ms delay
- Server 5: 300ms delay

There were 4 handlers on each one of these servers.
- `/foo`: returns "foo", can be cached, obeys the delays set on the server
- `/dynamic`: returns a random number, cannot be cached, obeys the delays set on the server + 50-150ms random delay
- `/cached`: returns "cached", can be cached, obeys the delays set on the server
- `/random-delay`: returns a random number, cannot be cached, does not obey the delays set on the server, returns with a delay between 100ms to 300ms

### Minotaur vs Nginx (Default Algorithm)
`/foo`
![foo](https://github.com/user-attachments/assets/1e1e188e-c741-4194-a4b5-90232f6edf95)

`/dynamic`
![dynamic](https://github.com/user-attachments/assets/1fb3c85d-5aa3-4208-97b8-d479c4b64e9c)

`/cached/item-99`
![cached](https://github.com/user-attachments/assets/187125cc-34aa-4a14-9163-0a8c21a4bc38)

`/random-delay`
![random](https://github.com/user-attachments/assets/1150fc1b-1223-401e-b74b-baff027f191a)

### Other Nginx Algorithms
#### Weighted Least Connections
`/foo`
TBA

`/dynamic`
TBA

`/cached/item-99`
TBA

`/random-delay`
TBA

#### IP Hash
`/foo`
TBA

`/dynamic`
TBA

`/cached/item-99`
TBA

`/random-delay`
TBA

#### Random
`/foo`
TBA

`/dynamic`
TBA

`/cached/item-99`
TBA

`/random-delay`
TBA

## Other Features

**SSL/TLS encryption:** acts as a SSL terminator, configurable to use your own SSL certificates

## How Minotaur Works

Now let us understand how Minotaur actually works and why it performs better than NGINX in some of the benchmarks above.
Minotaur decides the route of a request based on how well different backend servers are doing, this means that the server doing good will receive more requests than one doing poorly.
The way we decide which server is `better` than another is by calculating their `mean response times` using an Exponential Moving Average.
An exponential moving average is an average which gives more weight to recent data, we used this formula to calculate the next average of a server.
```go
const alpha = 0.5 // Smoothing factor for Exponential Moving Average (EMA)
if server.TotalResponses == 0 {
    // Init
    server.AvgResponseMs = responseTime
} else {
    server.AvgResponseMs = int64(float64(server.AvgResponseMs)*(1-alpha) + float64(responseTime)*alpha)
}
server.TotalResponses++
```
I found 0.5 to work best for the value of alpha.

Then the proxy updates the weights of every server every 2 seconds.
```go
const smoothingFactor = 50 // Add to all response times for fairness
for i := range p.servers {
    server := &p.servers[i]
    if server.AvgResponseMs == 0 {
        server.AvgResponseMs = 1 // Divide by zero prevention
    }
    server.Weight = int(1000 / (server.AvgResponseMs + smoothingFactor))
    if server.Weight < 1 {
    server.Weight = 1
    }
}
```
## Prerequisites
Before we begin with an example usage, please ensure you have the following installed:
1. [Go](https://go.dev/doc/install)
2. [Docker](https://docs.docker.com/engine/install/) & [Docker Compose](https://docs.docker.com/compose/install/)

## Getting Started

#### Clone the Repository:
```bash
git clone https://github.com/rudransh-shrivastava/minotaur
```

#### Change Directory
```bash
cd minotaur
```

#### Install Dependencies
```bash
go mod tidy
```

#### Get a self-signed SSL Certificate
For local development and testing, you can use [mkcert](https://github.com/FiloSottile/mkcert) to generate a self-signed SSL certificate. This is a simple tool that creates locally trusted certificates for development purposes.
```bash
mkcert -install
mkcert localhost
```
This generates a `localhost.pem` (the SSL certificate) and `localhost-key.pem` (the private key) in the project root

#### Setup Environment Variables

create a `.env` file.
```bash
touch .env
```

**Available Environment Variables:**
```
PORT=443
LOAD_BALANCING_MODE="WEIGHTED_ROUND_ROBIN"
SERVERS="http://localhost:8081,http://localhost:8082,http://localhost:8083"
REDIS_HOST="localhost:6379"
SSL_KEY_PATH="localhost-key.pem"
SSL_CERT_PATH="localhost.pem"
```

**Descriptions:**

 - *PORT*: Port on which the reverse proxy server will listen for incoming HTTPS requests.
 -  *LOAD_BALANCING_MODE*: The load balancing algorithm used by the proxy to distribute incoming requests across backend servers. Available modes:
	 -   `WEIGHTED_ROUND_ROBIN`: Distributes requests based on the server weights. Servers with higher weights will receive more requests.
	-   `ROUND_ROBIN`: Distributes requests evenly across all servers.
 - *SERVERS*: the list of backend servers to which the proxy will forward requests. Each server should be specified by its URL separated by a `,`
 - *REDIS_HOST*: Address of Redis for caching.
 - *SSL_KEY_PATH*: The path to the private SSL key file.
 - *SSL_CERT_PATH*: Path to the SSL certificate file.

You can copy these default environment variables to your `.env` file. If you don't create a `.env` file, these defaults will be used by the server.

## Usage

#### Run Testing Servers and Redis
we will use docker-compose to run the testing servers and a redis instance.

**build the containers**
```bash
docker-compose build
```
if you get a permission denied error: use `sudo`
```bash
sudo docker-compose build
```

**run the containers**
```bash
docker-compose up -d
```
This runs the testing servers(3) and a redis instance for caching.

#### Run the Reverse Proxy
Now, its time to run the reverse proxy, the reverse proxy will act as an intermediary between the servers and the clients

**Build Minotaur:**
if you have `make` installed you can:
```bash
make build
```
if you don't then you can:
```bash
go build -o bin/minotaur
```
**Run Minotaur**
```bash
bin/minotaur
```
if you get the error : `Error starting server: listen tcp :443: bind: permission denied`
use `sudo` to run Minotaur
```bash
sudo bin/minotaur
```

## License

This project is Licensed under the MIT License.
