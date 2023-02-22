FROM harbor.yunss.com:5000/base/base:latest
ADD srv /sip/srv
ADD config.yml /sip/config.yml
WORKDIR /sip
ENTRYPOINT [ "./srv" ]