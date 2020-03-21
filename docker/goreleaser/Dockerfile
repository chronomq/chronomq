# Chronomq is a job-queue that orders jobs by their trigger time.
#
# docker run --rm -it \
# 			 --name chronomq \
#				 chronomq/chronomq

FROM scratch as binBase
COPY chronomq /

FROM alpine:edge
LABEL maintainer="Urjit Singh Bhatia<(urjitsinghbhatia@gmail.com> (github: @chronomq)"

ARG CHRONOMQ_VERSION=0.1.0

WORKDIR /usr/local/bin/

RUN     apk --no-cache add \
    ca-certificates

COPY --from=binBase /chronomq /usr/local/bin/chronomq
ENTRYPOINT [ "chronomq" ]
