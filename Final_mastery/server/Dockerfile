FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 go build -o /server .

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=build /server /server
ENTRYPOINT ["/server"]
