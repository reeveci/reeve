FROM golang AS builder

ARG gitToken
RUN echo machine github.com login git password $gitToken >$HOME/.netrc

COPY . /app
WORKDIR /app/reeve-server

ENV GOFLAGS="-buildvcs=false"
ENV CGO_ENABLED=0
RUN go build -ldflags "-X main.buildVersion=`sh ../version.sh`" -o /usr/local/bin/reeve-server .

FROM alpine

COPY --chmod=755 --from=builder /usr/local/bin/reeve-server /usr/local/bin/
RUN mkdir -p /etc/reeve/plugins

ENV REEVE_PLUGIN_DIRECTORY=/etc/reeve/plugins
ENV REEVE_HTTP_PORT=9080
ENV REEVE_HTTPS_PORT=9443
ENV REEVE_TLS_CERT_FILE=
ENV REEVE_TLS_KEY_FILE=

ENV REEVE_MESSAGE_SECRETS=
ENV REEVE_CLI_SECRETS=
ENV REEVE_WORKER_SECRETS=
ENV REEVE_WORKER_GROUPS=

EXPOSE 9080 9443
CMD ["reeve-server"]
