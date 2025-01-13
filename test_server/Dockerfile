FROM golang:1.23

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN go build -o /test_server

CMD ["/test_server"]