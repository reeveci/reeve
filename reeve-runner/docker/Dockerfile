FROM golang AS builder

COPY . /app
WORKDIR /app/reeve-runner

ENV GOFLAGS="-buildvcs=false"
ENV CGO_ENABLED=0
RUN go build -ldflags "-X main.buildVersion=`sh ../version.sh`" -o /usr/local/bin/reeve-runner .

WORKDIR /app/reeve-tools

ENV GOFLAGS="-buildvcs=false"
ENV CGO_ENABLED=0
RUN go build -ldflags "-X main.buildVersion=`sh ../version.sh`" -o /usr/local/bin/reeve-tools .

FROM docker

COPY --chmod=755 --from=builder /usr/local/bin/reeve-runner /usr/local/bin/
COPY --chmod=755 --from=builder /usr/local/bin/reeve-tools /usr/local/bin/
COPY --chmod=755 reeve-runner/docker/docker-entrypoint.sh /usr/local/bin/

# docker config
ENV DOCKER_LOGIN_REGISTRIES=
#ENV DOCKER_LOGIN_REGISTRY=
#ENV DOCKER_LOGIN_USER=
#ENV DOCKER_LOGIN_PASSWORD=

# runner config
ENV REEVE_API_PORT=80
ENV REEVE_RUNTIME_ENV=docker
ENV REEVE_DOCKER_COMMAND=docker
ENV REEVE_FORWARD_PROXY=true
ENV REEVE_NO_DESCRIPTION=

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["reeve-runner"]
