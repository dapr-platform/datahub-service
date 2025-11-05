FROM golang:1.23.1-alpine3.19 AS build
RUN  apk add --no-cache git upx \
    && rm -rf /var/cache/apk/* \
    && rm -rf /root/.cache \
    && rm -rf /tmp/*
RUN mkdir /app
WORKDIR /app
COPY go.mod .
COPY go.sum .
ENV GOSUMDB=off
RUN go mod tidy
COPY . .
RUN go build -ldflags "-s -w" -o  datahub-service && upx -9 datahub-service

FROM alpine:3.19
RUN  apk add --no-cache tzdata && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && rm -rf /var/cache/apk/* \
    && rm -rf /root/.cache \
    && rm -rf /tmp/*

    
RUN mkdir /app
WORKDIR /app
COPY --from=build /app/datahub-service .

EXPOSE 80
CMD ["sh","-c","./datahub-service"]
