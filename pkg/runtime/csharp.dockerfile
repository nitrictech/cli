FROM mcr.microsoft.com/dotnet/sdk:7.0-alpine3.16-arm64v8 AS build
WORKDIR /app

# Copy everything
COPY . ./
# Build and publish a release
RUN dotnet publish -c Release -o out

# Build runtime image
FROM mcr.microsoft.com/dotnet/aspnet:7.0.0-alpine3.16-arm64v8

ARG HANDLER
ENV HANDLER=${HANDLER}

WORKDIR /app
COPY --from=build /app/out .
ENTRYPOINT ["sh", "-c", "./$HANDLER"]