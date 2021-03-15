FROM scratch
COPY beacon /
EXPOSE 8080
ENTRYPOINT ["/beacon"]