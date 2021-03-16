FROM alpine:edge as base

RUN apk add -U --no-cache ca-certificates


FROM scratch

COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY beacon /

EXPOSE 8080
ENTRYPOINT ["/beacon"]