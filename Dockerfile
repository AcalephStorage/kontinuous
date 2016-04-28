FROM golang:1.6

ENV GOPATH /go
ENV SWAGGER_UI /swagger/dist

ADD . /go/src/github.com/AcalephStorage/kontinuous
WORKDIR /go/src/github.com/AcalephStorage/kontinuous

RUN mkdir /swagger && tar xvzf third_party/swagger.tar.gz -C /swagger

# create and remove downloaded libraries
RUN make && rm -rf /go/bin && rm -rf /go/lib

ENTRYPOINT build/bin/kontinuous 
