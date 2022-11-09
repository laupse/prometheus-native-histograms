FROM golang:1.19.2-alpine3.16 as build

ENV GO111MODULE=on

WORKDIR /go/src/github.com/laupse/native_histograms
COPY . .

RUN go build -o /go/bin/native_histograms

FROM alpine:3.16

COPY --from=build /go/bin/native_histograms /go/bin/

CMD [ "/go/bin/native_histograms" ]