FROM loads/alpine:3.8
ENV WORKDIR=/app
ADD ./temp/linux_amd64/main $WORKDIR/main
RUN chmod +x $WORKDIR/main
WORKDIR $WORKDIR
CMD ["./main"]
