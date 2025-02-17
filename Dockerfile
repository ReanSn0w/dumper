FROM golang:1.22-alpine AS application

ARG TAG
ADD . /bundle

WORKDIR /bundle

RUN apk --no-cache add ca-certificates

RUN \
  revision=${TAG} && \
  echo "Building container. Revision: ${revision}" && \
  go build -ldflags "-X main.revision=${revision}" -o /srv/app ./cmd/main.go

# Финальная сборка образа
FROM scratch
COPY --from=application /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=application /srv /srv

ENV PORT=8080
ENV DISABLED=false
ENV BODY=false
ENV TARGET=

EXPOSE 8080
VOLUME [ "/data" ]
WORKDIR /srv
ENTRYPOINT ["/srv/app"]
