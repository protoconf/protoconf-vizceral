FROM node:alpine

# Create app directory
WORKDIR /usr/src/app

# Install app dependencies
COPY package.json webpack.config.js yarn.lock .babelrc .eslintrc .jshintrc ./
RUN yarn install

# Bundle app source
COPY app app

RUN yarn run build
RUN yarn run copy:fonts
RUN yarn run copy:json

FROM golang:1.16
WORKDIR /go/src/github.com/protoconf/protoconf-vizceral
RUN go get -u github.com/go-bindata/go-bindata/...

COPY --from=0 /usr/src/app/dist dist

COPY *.go ./
COPY pkg/ pkg
RUN find .; go mod init && go mod tidy
RUN go-bindata -o ./pkg/static/bindata.go -pkg static -fs -prefix="dist/" dist/... 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=1 /go/src/github.com/protoconf/protoconf-vizceral/app .
COPY --from=1 /go/src/github.com/protoconf/protoconf-vizceral .
# COPY --from=0 /usr/src/app/dist dist
ENTRYPOINT ["./app"]
