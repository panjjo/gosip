FROM harbor.yunss.com:5000/base/base:latest
ADD srv /
ADD config.yml /
ENTRYPOINT [ "/srv" ]