# syntax=docker/dockerfile:1

# Build the application from source
FROM golang:1.19 AS build-stage

# set working directory
WORKDIR /app

# copy the mod file and download dependancies
COPY go.mod go.sum ./
RUN go mod download

# copy all code files
COPY *.go ./

# compile the program out into the WORKDIR. params: CGO_ENABLED = 'c language go', GOOS = 'go operating system'
RUN CGO_ENABLED=0 GOOS=linux go build -o /http_api_svr

# signal to the
CMD ["/http_api_svr"]