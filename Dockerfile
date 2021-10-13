FROM golang:1.17.2-alpine

WORKDIR /go/src/regonize
COPY cmd/regonize .
RUN go build
ENTRYPOINT [ "/go/src/regonize/regonize" ]
EXPOSE 8080