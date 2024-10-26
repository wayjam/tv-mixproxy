FROM golang as builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 make build

FROM gcr.io/distroless/base-debian11 AS release
RUN touch /app/config.yaml
COPY --from=builder /app/build/tv-mixproxy /app/tv-mixproxy
WORKDIR /app
CMD ["/app/tv-mixproxy", "--config", "/app/config.yaml"]