FROM alpine:3.14

ADD ./bin/gwyneth /usr/local/bin/gwyneth
ADD ./http/templates /usr/local/src/http/templates
ADD ./http/static /usr/local/src/http/static

RUN mkdir /app

CMD ["/usr/local/bin/gwyneth", "-c", "/app/gwyneth.yaml"]
