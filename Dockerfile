FROM ubuntu:latest

COPY pod-mutating-webhook /pod-mutating-webhook

ENTRYPOINT [ "./pod-mutating-webhook" ]
