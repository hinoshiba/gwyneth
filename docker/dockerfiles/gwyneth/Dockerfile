FROM alpine:3.14

ADD ./bin/gwyneth /usr/local/bin/gwyneth

RUN mkdir /app

CMD ["/usr/local/bin/gwyneth", "-c", "/app/gwyneth.yaml"]
