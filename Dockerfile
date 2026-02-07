FROM alpine:latest
ARG TARGETPLATFORM
COPY $TARGETPLATFORM/karmafun /usr/local/bin/config-function
CMD ["config-function"]
