FROM golang AS builder

COPY . /app
WORKDIR /app/reeve-worker

ENV GOFLAGS="-buildvcs=false"
ENV CGO_ENABLED=0
RUN go build -o /usr/local/bin/reeve-worker .

FROM docker

COPY --chmod=755 --from=builder /usr/local/bin/reeve-worker /usr/local/bin/
COPY --chmod=755 reeve-worker/docker/docker-entrypoint.sh /usr/local/bin/
COPY --chmod=755 reeve-worker/docker/docker-runner.sh /usr/local/bin/

# docker config
ENV DOCKER_LOGIN_REGISTRY=
ENV DOCKER_LOGIN_USER=
ENV DOCKER_LOGIN_PASSWORD=

# worker config
ENV REEVE_SERVER_API=http://reeve-server:9080
ENV REEVE_WORKER_AUTH_HEADER=Authorization
ENV REEVE_WORKER_AUTH_PREFIX="Bearer "
ENV REEVE_WORKER_SECRET=
ENV REEVE_WORKER_GROUP=
ENV REEVE_RUNNER_COMMAND=/usr/local/bin/docker-runner.sh
ENV REEVE_RUNNER_IMAGE=

# runner config
ENV REEVE_API_PORT=80
ENV REEVE_RUNTIME_ENV=docker
ENV REEVE_DOCKER_COMMAND=docker
ENV REEVE_FORWARD_PROXY=true
ENV REEVE_NO_DESCRIPTION=

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["reeve-worker"]
