FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 go build -o kvstore .

FROM alpine:3.19
COPY --from=build /app/kvstore /kvstore
EXPOSE 8080
CMD ["/kvstore"]
