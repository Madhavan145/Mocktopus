FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY bin/gateway /bin/gateway
EXPOSE 8080
ENTRYPOINT ["/bin/gateway"]
