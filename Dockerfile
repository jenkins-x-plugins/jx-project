FROM centos:7

RUN yum install -y git

ENTRYPOINT ["jx-project"]

COPY ./build/linux/jx-project /usr/bin/jx-project