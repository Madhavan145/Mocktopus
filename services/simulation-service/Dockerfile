FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY bin/simulation /bin/simulation
EXPOSE 8084
ENTRYPOINT ["/bin/simulation"]
