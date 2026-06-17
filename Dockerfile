FROM golang:1.26

WORKDIR /app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download

# copy and build application
COPY . .
RUN go build .

EXPOSE 8080

CMD ["/bpo"]
