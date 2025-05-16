FROM cgr.dev/chainguard/go AS builder

WORKDIR /app

COPY . .

RUN make build

FROM cgr.dev/chainguard/wolfi-base:latest

COPY --from=builder /app/bin/nanobot .

CMD ["./nanobot"]