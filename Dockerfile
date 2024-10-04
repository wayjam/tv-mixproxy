FROM golang as builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 make build

FROM gcr.io/distroless/base-debian11 AS release
COPY --from=builder /app/build/tvbox-mixproxy /app/tvbox-mixproxy
WORKDIR /app
CMD ["/app/tvbox-mixproxy"]