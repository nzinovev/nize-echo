# syntax=docker/dockerfile:1
# The nize-echo image: the admin build, then the Go binary, in a small
# runtime image. Build from the repository root:
#   docker build -f deploy/Dockerfile .

FROM node:24-alpine AS admin
WORKDIR /src/admin
COPY admin/package.json admin/package-lock.json ./
RUN npm ci
COPY admin/ ./
RUN npm run build

FROM golang:1.25-alpine AS server
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# The admin build sits where the Go embed will read it.
COPY --from=admin /src/admin/dist ./admin/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /out/site ./cmd/site

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=server /out/site /site
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/site"]
