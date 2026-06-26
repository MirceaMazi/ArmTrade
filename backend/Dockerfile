FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o armtrade-backend .

# distroless/static has no package manager — nothing to fetch at build time.
# The Go binary is fully static (CGO_ENABLED=0) so no OS libs are needed.
FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /app/armtrade-backend .
EXPOSE 8080
CMD ["/app/armtrade-backend"]
