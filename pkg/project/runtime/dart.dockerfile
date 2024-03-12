FROM dart:stable AS build

ARG HANDLER
WORKDIR /app

# Resolve app dependencies.
COPY pubspec.* ./
RUN dart pub get

# Ensure the ./bin folder exists
RUN mkdir -p ./bin

# Copy app source code and AOT compile it.
COPY . .
# Ensure packages are still up-to-date if anything has changed
RUN dart pub get --offline
RUN dart compile exe ./${HANDLER} -o bin/main

# Build a minimal serving image from AOT-compiled `/server` and required system
# libraries and configuration files stored in `/runtime/` from the build stage.
FROM alpine

COPY --from=build /runtime/ /
COPY --from=build /app/bin/main /app/bin/

ENTRYPOINT ["/app/bin/main"]
