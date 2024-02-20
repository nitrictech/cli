FROM mcr.microsoft.com/dotnet/sdk:7.0 AS build

# https://github.com/dotnet/runtime/issues/94909
ENV DOTNET_EnableWriteXorExecute=0

ARG HANDLER

WORKDIR /app

# Copy everything
COPY . ./

# Build and publish a release
RUN dotnet publish -c Release -o out --self-contained -p:PublishSingleFile=true -p:PublishTrimmed=true

# Build runtime image
FROM mcr.microsoft.com/dotnet/runtime-deps:7.0

ARG HANDLER

COPY --from=build /app/out/${HANDLER} /usr/bin/handler

ENTRYPOINT ["handler"]