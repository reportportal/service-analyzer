FROM scratch

MAINTAINER Ivan Sharamet <ivan_sharamet@epam.com>

ADD ./bin/service-analyzer /

EXPOSE 8080
ENTRYPOINT ["/service-analyzer"]