FROM alpine:3.4

ENV GOPATH /go
ENV SWAGGER_UI /swagger/dist

ADD . /go/src/github.com/AcalephStorage/kontinuous
WORKDIR /go/src/github.com/AcalephStorage/kontinuous

RUN mkdir /swagger && tar xvzf third_party/swagger.tar.gz -C /swagger

# create and remove downloaded libraries
RUN apk update && \
    apk add make git go ca-certificates && \
    make && \
    mv build/bin/kontinuous /bin && \
    mv build/bin/kontinuous-cli /bin && \
    apk del --purge git && \
    rm -rf /var/cache/apk/*

EXPOSE 3005

ENTRYPOINT /bin/kontinuous
