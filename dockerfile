FROM golang:1.12.5-alpine3.9 AS API-Builder

WORKDIR /

# Add ca-certificates to get the proper certs for making requests,
# gcc and musl-dev for any cgo dependencies, and
# git for getting dependencies residing on github
RUN apk update && \
    apk add --no-cache ca-certificates gcc git musl-dev

# Get our dependency managing files
COPY go.mod go.sum ./

# Install all of the nessesary dependencies
RUN go mod download

# Copy in our Go files
COPY ./functionality ./functionality
COPY ./api-server.go .

# Compile the program statically with local dependencies
RUN env CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -a -v -o api-server

# Last stage of build, adding in files and running
# newly compiled webserver
FROM scratch

# Copy the Go program compiled in the second stage
COPY --from=API-Builder /api-server /

# Add HTTPS Certificates for making HTTP requests from the webserver
COPY --from=API-Builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Expose ports 80 to host machine
EXPOSE 80

# Run program
ENTRYPOINT ["/api-server"]
