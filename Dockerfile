FROM golang:1.6-alpine

ENV GOPATH /go
ENV SWAGGER_UI /swagger/dist

ADD . /go/src/github.com/AcalephStorage/kontinuous
WORKDIR /go/src/github.com/AcalephStorage/kontinuous

RUN mkdir /swagger && tar xvzf third_party/swagger.tar.gz -C /swagger

# create and remove downloaded libraries
RUN apk update && \
    apk add make git && \
    make && \
    rm -rf /go/bin && \
    rm -rf /go/lib && \
    apk del --purge make git && \
    rm -rf /var/cache/apk/*

EXPOSE 3005

ENTRYPOINT build/bin/kontinuous
