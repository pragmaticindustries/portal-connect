FROM golang:latest as builder

ENV PATH /go/bin:/usr/local/go/bin:$PATH
ENV GOPATH /go

COPY . /go/srcgithub.com/pragmatic-industries/portal-connect

RUN set -x \
    && cd /go/srcgithub.com/pragmatic-industries/portal-connect \
    && go build \
    && mv portal-connect /usr/bin/portal-connect \
    && rm -rf /go \
    && echo "Build complete."

FROM ubuntu:latest

COPY --from=builder /usr/bin/portal-connect /usr/bin/portal-connect

#ENV PORT 8080
#EXPOSE $PORT

ENTRYPOINT [ "portal-connect" ]
CMD [ "" ]