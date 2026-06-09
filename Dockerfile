FROM golang:1.26

WORKDIR /app

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download

# copy and build application
COPY . .
RUN go build -v -o /bin/bpo .

EXPOSE 8080

CMD ["ls /bin"]
CMD ["ls /bin/bpo"]
CMD ["/bin/bpo"]
