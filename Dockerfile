FROM ghcr.io/jenkins-x/jx-boot:latest

ENTRYPOINT ["jx-project"]

COPY ./build/linux/jx-project /usr/bin/jx-project
