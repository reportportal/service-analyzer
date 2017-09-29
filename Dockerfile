FROM golang:1.9

WORKDIR /go/src/github.com/reportportal/service-analyzer/
ARG version

## Copy makefile and glide before to be able to cache vendor
COPY Makefile ./
COPY glide.yaml ./
COPY glide.lock ./

RUN make get-build-deps
COPY ./ ./
RUN make build version=$version

FROM alpine:latest
ARG service
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/github.com/reportportal/service-analyzer/bin/service-analyzer ./app
CMD ["./app"]