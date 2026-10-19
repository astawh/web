FROM --platform=$BUILDPLATFORM golang:1.27.2-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS builder
WORKDIR /src
COPY go.mod ./
COPY main.go application.go ./
COPY internal ./internal
COPY model ./model
COPY site ./site
COPY view ./view
ARG TARGETOS TARGETARCH
ARG VERSION=dev
ARG REVISION
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Revision=${REVISION}" -o /out/web .

FROM scratch
COPY --from=builder /out/web /web
USER 1001:1001
EXPOSE 8080
ENTRYPOINT ["/web"]
CMD ["-addr", "0.0.0.0:8080"]
