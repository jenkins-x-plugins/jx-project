FROM centos:7

RUN yum install -y git

ENTRYPOINT ["jwizard"]

COPY ./build/linux/jwizard /usr/bin/jwizard