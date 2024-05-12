FROM alpine:3.14

ADD ./bin/gwyneth /usr/local/bin/gwyneth
ADD ./http/templates /usr/local/src/http/templates
ADD ./http/static /usr/local/src/http/static
ADD ./scripts /var/service/gwyneth/bin

RUN apk add bash && \
    chmod u+x /var/service/gwyneth/bin/*

WORKDIR "/var/service/gwyneth"

CMD ["/usr/local/bin/gwyneth", "-c", "/app/gwyneth.yaml"]
