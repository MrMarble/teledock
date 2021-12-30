ARG GO_VERSION=1.17

## Build container
FROM golang:${GO_VERSION}-alpine AS builder

RUN apk add --no-cache ca-certificates git zip

WORKDIR /src

COPY ./go.mod ./go.sum ./
RUN go mod download

COPY ./ ./
RUN CGO_ENABLED=0 go build -installsuffix 'static' -o /teledock /src/cmd/teledock

## Final container
FROM scratch AS final

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /teledock /teledock

ENTRYPOINT ["/teledock"]
