FROM golang:1.24-alpine

ARG BUILD_NAME
ARG SERVICE_PORT
WORKDIR /usr/src/app

# Pre-copy/cache go.mod for pre-downloading dependencies
COPY go.mod go.sum ./
COPY shared ./shared
COPY ${BUILD_NAME} ./${BUILD_NAME}
ENV SERVICE_NAME=${BUILD_NAME}
ENV PORT=${SERVICE_PORT}


RUN go mod tidy && go mod download

RUN go build -v -o /usr/local/bin/app ./${BUILD_NAME}

CMD ["app"]