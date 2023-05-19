# syntax=docker/dockerfile:1
FROM node:alpine as build

ARG HANDLER

WORKDIR /usr/app

# Python and make are required by certain native package build processes in NPM packages.
ENV PYTHONUNBUFFERED=1
RUN apk add --update --no-cache python3 make g++ && ln -sf python3 /usr/bin/python
RUN python3 -m ensurepip
RUN pip3 install --no-cache --upgrade pip setuptools

RUN yarn global add typescript @vercel/ncc

COPY . .

RUN yarn import || echo Lockfile already exists

RUN set -ex; yarn install --frozen-lockfile --cache-folder /tmp/.cache; rm -rf /tmp/.cache;

RUN test -f tsconfig.json || echo "{\"compilerOptions\":{\"esModuleInterop\":true,\"target\":\"es2015\",\"moduleResolution\":\"node\"}}" > tsconfig.json

# make prisma external to bundle - https://github.com/prisma/prisma/issues/16901#issuecomment-1362940774 \
# TODO: remove when custom dockerfile support is available
RUN ncc build ${HANDLER} -o lib/ -e .prisma/client -e @prisma/client

FROM node:alpine as final

WORKDIR /usr/app

RUN apk update && \
    apk add --no-cache ca-certificates && \
    update-ca-certificates

COPY --from=build /usr/app/lib/ ./lib/

COPY . .

RUN set -ex; \
    yarn install --production --frozen-lockfile --cache-folder /tmp/.cache; \
    rm -rf /tmp/.cache; \
    # prisma fix for docker installs: https://github.com/prisma/docs/issues/4365
    # TODO: remove when custom dockerfile support is available
    test -d ./prisma && npx prisma generate || echo "";


ENTRYPOINT ["node", "lib/index.js"]