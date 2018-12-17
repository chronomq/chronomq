# Goyaad is a job-queue that orders jobs by their trigger time.
#
# docker run --rm -it \
# 			 --name goyaad \
#				 urjitbhatia/goyaad

FROM alpine:edge
LABEL maintainer Urjit Singh Bhatia<(urjitsinghbhatia@gmail.com> (github: @urjitbhatia)

ARG GOYAAD_VERSION=0.1.0

WORKDIR /usr/local/bin/

RUN     apk --no-cache add \
    ca-certificates

RUN buildDeps=' \
    bash \
    ' \
    set -x \
    && apk --no-cache add $buildDeps \
    # Enable this to target a public release
    # && wget -O goyaad https://github.com/urjitbhatia/goyaad/releases/download/v${GOYAAD_VERSION}/goyaad_${GOYAAD_VERSION}_Linux_x86_64.tar.gz \
    # && chmod +x /usr/local/bin/goyaad \
    && apk del $buildDeps \
    && echo "Build complete."

# For dev mode, just use builds from dist
COPY dist/linux_amd64/goyaad /usr/local/bin/goyaad
ENTRYPOINT [ "goyaad" ]
