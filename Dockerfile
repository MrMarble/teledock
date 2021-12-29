FROM golang:1.17 AS builder
WORKDIR /source
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -o teledock ./cmd/teledock/main.go

FROM scratch
WORKDIR /bot/
COPY --from=builder /source/teledock .
CMD ["./teledock"]