FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY bin/auth /bin/auth
EXPOSE 8081
ENTRYPOINT ["/bin/auth"]
