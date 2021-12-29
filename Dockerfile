ARG GO_VERSION=1.17

## Build container
FROM golang:${GO_VERSION}-alpine AS builder

RUN mkdir /user && \
    echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
    echo 'nobody:x:65534:' > /user/group

RUN apk add --no-cache ca-certificates git zip

WORKDIR /src

COPY ./go.mod ./go.sum ./
RUN go mod download

COPY ./ ./
RUN CGO_ENABLED=0 go build -installsuffix 'static' -o /teledock /src/cmd/teledock

## Final container
FROM scratch AS final

COPY --from=builder /user/group /user/passwd /etc/
COPY --from=builder /teledock /teledock

USER 65534:65534

ENTRYPOINT ["/teledock"]