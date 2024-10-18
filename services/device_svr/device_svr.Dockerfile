# syntax=docker/dockerfile:1

# Build the application from source
FROM golang:1.23 AS build-stage

# set working directory
WORKDIR /app

# Copy shared packages
COPY ./pkg pkg

# In the docker compose file we set build context to the root dir of the project, which means we have to specify this service's path from the root of the project. 
# We copy the directory structure of the project on the host machine into the container as all imports are relative to the project structure, would break pkg imports otherwise.

# Copy mod and sum files into the the container. 
COPY ./services/device_svr/go.mod ./services/device_svr/go.sum /app/services/device_svr/

# copy all code files
COPY ./services/device_svr/*.go /app/services/device_svr/

WORKDIR /app/services/device_svr
RUN go mod download

# compile the program out into the WORKDIR. params: CGO_ENABLED = 'c language go', GOOS = 'go operating system'
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/device_svr.bin

# run the app
CMD ["/app/bin/device_svr.bin"]