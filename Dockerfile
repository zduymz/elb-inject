FROM alpine:3.9
LABEL maintainer="duymai"

COPY ./build/linux/elb-inject /bin/elb-inject

USER nobody

ENTRYPOINT ["/bin/elb-inject"]

