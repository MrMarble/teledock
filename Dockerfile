FROM golang:1.14 AS builder
WORKDIR /source
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -o teledock .

FROM scratch
WORKDIR /bot/
COPY --from=builder /source/teledock .
CMD ["./teledock"]