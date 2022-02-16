FROM golang:1.11-alpine AS build
WORKDIR /go/src/github.com/lyzs90/buildkit-pack
RUN apk add --no-cache file
ADD . .
RUN CGO_ENABLED=0 go build -o /out/pack ./cmd/pack && file /out/pack | grep "statically linked"
  
FROM scratch
COPY --from=build /out/pack /bin/pack
ENTRYPOINT ["/bin/pack"]