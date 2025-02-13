# syntax=docker/dockerfile:1

# Build the application from source
FROM golang:1.23 AS build-stage

# set working directory
WORKDIR /app

# Copy shared packages
COPY ./pkg pkg

# so we can concat files
RUN apt-get -y update
RUN apt-get -y upgrade
RUN apt-get install -y ffmpeg

# In the docker compose file we set build context to the root dir of the project, which means we have to specify this service's path from the root of the project. 
# We copy the directory structure of the project on the host machine into the container as all imports are relative to the project structure, would break pkg imports otherwise.

# Copy mod and sum files into the the container. 
COPY ./services/cam_svr/go.mod ./services/cam_svr/go.sum /app/services/cam_svr/

# copy all code files
COPY ./services/cam_svr/*.go /app/services/cam_svr/

# TODO remove, test file so we don't upload
COPY ./services/cam_svr/verysmallfile.txt /app/services/cam_svr/

WORKDIR /app/services/cam_svr
RUN go mod download

# compile the program out into the WORKDIR. params: CGO_ENABLED = 'c language go', GOOS = 'go operating system'
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/cam_svr.bin

# copy over and make our script/s executable
COPY ./services/cam_svr/concat.sh /app/services/cam_svr/
COPY ./services/cam_svr/convert.sh /app/services/cam_svr/
RUN chmod +x /app/bin/cam_svr.bin concat.sh
RUN chmod +x /app/bin/cam_svr.bin convert.sh

# run the app
CMD ["/app/bin/cam_svr.bin"]