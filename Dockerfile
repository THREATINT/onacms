FROM scratch
ADD onacms /
CMD ["/onacms", "--dir=/data"]
