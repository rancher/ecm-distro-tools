FROM alpine as builder

RUN apk update && \
    apk add make git build-base
COPY . /standup
RUN cd /standup && \
    make

FROM scratch
COPY --from=builder /standup/bin/standup /standup
ENTRYPOINT ["/standup"]
