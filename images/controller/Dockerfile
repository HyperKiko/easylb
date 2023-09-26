FROM golang AS builder

WORKDIR /usr/src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/controller/*.go .
COPY pkg pkg

RUN CGO_ENABLED=0 go build -o build/controller

FROM alpine

WORKDIR /

COPY --from=builder /usr/src/build/controller /controller

ENTRYPOINT ["/controller"]