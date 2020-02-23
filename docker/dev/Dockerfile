# Chronomq is a job-queue that orders jobs by their trigger time.
#
# docker run --rm -it \
# 			 --name chronomq \
#				 chronomq/chronomq

FROM alpine:edge
LABEL maintainer Urjit Singh Bhatia<(urjitsinghbhatia@gmail.com> (github: @chronomq)

ARG CHRONOMQ_VERSION=0.1.0

WORKDIR /usr/local/bin/

RUN     apk --no-cache add \
    ca-certificates

RUN buildDeps=' \
    bash \
    ' \
    set -x \
    && apk --no-cache add $buildDeps \
    # Enable this to target a public release
    # && wget -O chronomq https://github.com/chronomq/chronomq/releases/download/v${CHRONOMQ_VERSION}/chronomq_${CHRONOMQ_VERSION}_Linux_x86_64.tar.gz \
    # && chmod +x /usr/local/bin/chronomq \
    && apk del $buildDeps \
    && echo "Build complete."

# For dev mode, just use builds from dist
COPY dist/linux_amd64/chronomq /usr/local/bin/chronomq
ENTRYPOINT [ "chronomq" ]
