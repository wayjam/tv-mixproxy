FROM golang as builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 make build
RUN touch /app/build/config.yaml

FROM gcr.io/distroless/base-debian11 AS release
COPY --from=builder /app/build/tv-mixproxy /app/tv-mixproxy
COPY --from=builder /app/build/config.yaml /app/config.yaml
WORKDIR /app
CMD ["/app/tv-mixproxy", "--config", "/app/config.yaml"]