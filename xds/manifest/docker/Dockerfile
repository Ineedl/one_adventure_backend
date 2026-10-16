FROM alpine:3.20

WORKDIR /app
COPY temp/linux_amd64/main /app/main
RUN chmod 0755 /app/main

ENTRYPOINT ["/app/main"]
