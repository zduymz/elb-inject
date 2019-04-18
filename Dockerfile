FROM alpine:3.9
LABEL maintainer="duymai"

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY ./build/linux/elb-inject /bin/elb-inject

USER nobody

ENTRYPOINT ["/bin/elb-inject"]

