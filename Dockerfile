# deploy stage
# Ubuntu 22.04 must be the same as ubuntu during make release
FROM ubuntu:22.04

RUN apt-get -y update \
 && apt-get -y install \
    ca-certificates \
    curl \
 && rm -rf /var/lib/apt/lists/*

ARG VERSION=2022.9
ARG SERVER_VERSION=${VERSION}
# ARG CORTEZA_SERVER_PATH=https://releases.cortezaproject.org/files/corteza-server-${SERVER_VERSION}-linux-amd64.tar.gz
RUN mkdir /tmp/server
# ADD $CORTEZA_SERVER_PATH /tmp/server
COPY ./my-release/corteza-server-2022.9.0-dev.1-linux-amd64.tar.gz /tmp/server

VOLUME /data

RUN tar -zxvf "/tmp/server/corteza-server-2022.9.0-dev.1-linux-amd64.tar.gz" -C / && \
    rm -rf "/tmp/server" && \
    mv /corteza-server /corteza

WORKDIR /corteza

HEALTHCHECK --interval=30s --start-period=1m --timeout=30s --retries=3 \
    CMD curl --silent --fail --fail-early http://127.0.0.1:80/healthcheck || exit 1

ENV STORAGE_PATH "/data"
ENV CORREDOR_ADDR "corredor:80"
ENV HTTP_ADDR "0.0.0.0:80"
ENV HTTP_WEBAPP_ENABLED "false"
ENV PATH "/corteza/bin:${PATH}"

EXPOSE 80

ENTRYPOINT ["./bin/corteza-server"]

CMD ["serve-api"]
