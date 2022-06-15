FROM ubuntu
ADD srv /sip/srv
ADD config.yml /sip/config.yml
WORKDIR /sip
ENTRYPOINT [ "./srv" ]