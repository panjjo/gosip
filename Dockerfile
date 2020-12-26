FROM ubuntu
ADD srv /
ADD config.yml /
ENTRYPOINT [ "/srv" ]
