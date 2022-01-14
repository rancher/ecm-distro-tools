FROM alpine:3.14

RUN apk add ca-certificates

COPY bin/gen-release-notes /usr/local/bin

CMD ["gen-release-notes"]
