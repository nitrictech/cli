# syntax=docker/dockerfile:1
FROM node:22.4.1-alpine

ARG HANDLER
ENV HANDLER=${HANDLER}

# Python and make are required by certain native package build processes in NPM packages.
RUN --mount=type=cache,sharing=locked,target=/etc/apk/cache \
    apk --update-cache add git g++ make py3-pip

RUN apk update && \
    apk add --no-cache ca-certificates && \
    update-ca-certificates

COPY . .

RUN yarn import || echo Lockfile already exists

RUN \
  set -ex; \
  yarn install --production --frozen-lockfile --cache-folder /tmp/.cache; \
  rm -rf /tmp/.cache; \
  # prisma fix for docker installs: https://github.com/prisma/docs/issues/4365
  # TODO: remove when custom dockerfile support is available
  test -d ./prisma && npx prisma generate || echo "";

ENTRYPOINT node --import ./$HANDLER
