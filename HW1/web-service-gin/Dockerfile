# Use the official Golang image to create a build artifact.
FROM golang:1.23

# Set the working directory inside the container.
WORKDIR /app

# Copy the go.mod and go.sum files first (for better caching).
COPY go.mod go.sum ./

# Download dependencies.
RUN go mod download

# Copy the rest of the source code.
COPY *.go ./

# Build the application.
RUN go build -o /main

# Tell Docker that this container listens on port 8080.
EXPOSE 8080

# Command to run the executable.
CMD [ "/main" ]