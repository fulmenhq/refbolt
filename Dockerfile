FROM golang:1.25.5 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

ENV CGO_ENABLED=0

RUN go build \
    -ldflags="-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" \
    -o /out/refbolt \
    ./cmd/refbolt

FROM gcr.io/distroless/static-debian12

WORKDIR /app

COPY --from=builder /out/refbolt /usr/local/bin/refbolt

VOLUME ["/data/archive"]

ENTRYPOINT ["/usr/local/bin/refbolt"]
