FROM golang:1.11.3-alpine3.8 AS API-Builder

# Add ca-certificates to get the proper certs for making requests,
# gcc and musl-dev for any cgo dependencies, and
# git for getting dependencies residing on github
RUN apk update && \
    apk add --no-cache ca-certificates gcc git musl-dev

WORKDIR /go/src/github.com/the-rileyj/jetpack-api/

COPY ./api-server.go .

# Get dependencies locally, but don't install
# RUN go get -d -v ./...

###
COPY go.mod .

# Compile program statically with local dependencies
RUN env CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -a -v -o api-server

# Last stage of build, adding in files and running
# newly compiled webserver
FROM scratch

# Copy the Go program compiled in the second stage
COPY --from=API-Builder /go/src/github.com/the-rileyj/jetpack-api/ /

# Add HTTPS Certificates for making HTTP requests from the webserver
COPY --from=API-Builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Expose ports 80 to host machine
EXPOSE 80

# Run program
ENTRYPOINT ["/api-server"]