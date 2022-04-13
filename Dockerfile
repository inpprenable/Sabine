FROM golang:latest as builder
LABEL stage=builder
WORKDIR /app
ADD . /app
RUN CGO_ENABLED=0 GOOS=linux go build -o pbftnode

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/pbftnode /bin