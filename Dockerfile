FROM alpine:edge@sha256:e64a0b2fc7ff870c2b22506009288daa5134da2b8c541440694b629fc22d792e as base

RUN apk add -U --no-cache ca-certificates


FROM scratch

COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY beacon /

EXPOSE 8080
ENTRYPOINT ["/beacon"]