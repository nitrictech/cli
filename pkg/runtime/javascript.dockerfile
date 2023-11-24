# syntax=docker/dockerfile:1
FROM node:alpine as build

# Python and make are required by certain native package build processes in NPM packages.
RUN --mount=type=cache,sharing=locked,target=/var/cache/apk \
    apk --update-cache add git g++ make py3-pip

RUN yarn global add clean-modules

WORKDIR /usr/app

COPY package.json *.lock *-lock.json ./

RUN yarn import || echo ""

RUN --mount=type=cache,sharing=locked,target=/tmp/.yarn_cache \
    set -ex && \
    yarn install --production --prefer-offline --frozen-lockfile --cache-folder /tmp/.yarn_cache && \
    # cleanup / prune modules
    clean-modules -y

COPY . .

FROM node:alpine as final

ARG HANDLER
ENV HANDLER=${HANDLER}

RUN apk update && \
    apk add --no-cache ca-certificates && \
    update-ca-certificates

WORKDIR /usr/app

COPY . .

COPY --from=build /usr/app/node_modules/ ./node_modules/

COPY --from=build /usr/app/${HANDLER} ./${HANDLER}

# prisma fix for docker installs: https://github.com/prisma/docs/issues/4365
# TODO: remove when custom dockerfile support is available
RUN test -d ./prisma && npx prisma generate || echo "";

ENTRYPOINT node $HANDLER