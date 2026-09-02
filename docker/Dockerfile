# ---------- Builder ----------
FROM golang:1.25.0 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION

WORKDIR /app/cmd/summit
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags "-s -w -X github.com/RhykerWells/Summit/common.VERSION=$VERSION"

# ---------- Runtime ----------
FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder /app/cmd/summit/summit .

EXPOSE 80

CMD ["/app/summit"]