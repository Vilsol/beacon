FROM alpine:edge@sha256:2f77b6664f181b246244f9cd052155e74fb3f26d2a502edecd5fff0fc4bda388 as base

RUN apk add -U --no-cache ca-certificates


FROM scratch:latest@undefined

COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY beacon /

EXPOSE 8080
ENTRYPOINT ["/beacon"]