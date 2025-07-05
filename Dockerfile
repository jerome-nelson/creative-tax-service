FROM golang:1.24-alpine

ARG BUILD_NAME
ARG SERVICE_PORT

ENV SERVICE_NAME=${BUILD_NAME}
ENV PORT=${SERVICE_PORT}


WORKDIR /usr/src/app

# Copy shared utils
COPY shared ./shared

# Copy *.go source into folder
RUN mkdir -p ${BUILD_NAME}
COPY ${BUILD_NAME}/*.go ./${BUILD_NAME}

# Pre-copy/cache go.mod for pre-downloading dependencies
COPY go.mod go.sum ./
RUN go mod tidy && go mod download

RUN go build -v -o /usr/local/bin/app ./${BUILD_NAME}

CMD ["app"]