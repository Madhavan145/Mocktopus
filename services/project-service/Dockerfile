FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY bin/project /bin/project
EXPOSE 8082
ENTRYPOINT ["/bin/project"]
