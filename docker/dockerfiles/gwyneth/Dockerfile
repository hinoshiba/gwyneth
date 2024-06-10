FROM python:3

ADD ./bin/gwyneth /usr/local/bin/gwyneth
ADD ./http/templates /usr/local/src/http/templates
ADD ./http/static /usr/local/src/http/static
ADD ./scripts /app/bin

RUN chmod u+x /app/bin/*

RUN pip3 install requests

WORKDIR "/app/bin"

CMD ["/usr/local/bin/gwyneth", "-c", "/app/gwyneth.yaml"]
