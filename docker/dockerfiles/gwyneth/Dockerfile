FROM alpine:3.14

RUN apk add bash

ADD ./bin/gwyneth /usr/local/bin/gwyneth
ADD ./http/templates /usr/local/src/http/templates
ADD ./http/static /usr/local/src/http/static
ADD ./scripts /var/service/gwyneth/bin

RUN chmod u+x /var/service/gwyneth/bin/*

WORKDIR "/var/service/gwyneth"

CMD ["/usr/local/bin/gwyneth", "-c", "/app/gwyneth.yaml"]
