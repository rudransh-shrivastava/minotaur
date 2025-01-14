# Minotaur

Minotaur is a reverse proxy that supports load balancing, SSL/TLS encryption, caching with Redis, and dynamic server weight adjustments based on server response times.

**Load balancing:** distributes incoming requests using a weighted round robin algorithm (default/configurable)

**SSL/TLS encryption:** acts as a SSL terminator, configurable to use your own SSL certificates

**Caching:** uses Redis for caching HTTP responses to reduce server load.

# Prerequisites 
Before we begin with an example usage, please ensure you have the following installed:
1. [Go](https://go.dev/doc/install)
2. [Docker](https://docs.docker.com/engine/install/) & [Docker Compose](https://docs.docker.com/compose/install/)

# Example Usage

### Clone the Repository:
```bash
git clone https://github.com/rudransh-shrivastava/minotaur
```

### Change Directory
```bash
cd minotaur
```

### Install Dependencies
```bash
go mod tidy
```

### Get a self-signed SSL Certificate
For local development and testing, you can use [mkcert](https://github.com/FiloSottile/mkcert) to generate a self-signed SSL certificate. This is a simple tool that creates locally trusted certificates for development purposes.
```bash
mkcert -install
mkcert localhost
```
This generates a `localhost.pem` (the SSL certificate) and `localhost-key.pem` (the private key) in the project root

### Setup Environment Variables

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

### Run Testing Servers and Redis
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

### Run the Reverse Proxy
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

## How Load Balancing Works

Minotaur uses a **weighted round-robin** algorithm to distribute requests, requests are distributed across backend servers based on the calculated weight of each server. 

The weight of each server is adjusted periodically and dynamically based on its **average response time**, this ensures that faster servers handle more requests.

### Response Time Calculation and Weight Adjustment
The weight of each server is recalculated periodically based on its **average response time**. This is done using the **Exponential Moving Average (EMA)** formula.

The **Exponential Moving Average** gives higher weight to recent response times so that the proxy can quickly adapt to the changes in server performance.
The formula:
```
new_avg_response_time = (old_avg_response_time * (1 - alpha)) + (current_response_time * alpha)
```
where `alpha` is the smoothing factor (between 0 and 
1)

Then each servers weight is calculated using this formula:
```
server_weight = 1000 / (avg_response_time + smoothing_factor)
```
A server with lower average response time will have a higher weight and will handle more requests.

## Caching

Minotaur uses **Redis** for caching HTTP responses to reduce the load on backend servers. The proxy will check if a response for a given request is already cached. If it is, it serves the cached response. If not, it forwards the request to a backend server, caches the response, and then serves it.
