![](https://raw.githubusercontent.com/Netflix/vizceral/master/logo.png)

# Vizceral Example

This is a [protoconf](https://protoconf.github.io/protoconf) based application using Netflix's [vizceral](https://github.com/Netflix/vizceral) tool to visualize connections between services configured by `protoconf`.
In the future, we will integrate monitoring tools so you can visualize your entire production system and it's state using this tool.

# Prerequisites

1. Docker
2. [Protoconf](https://protoconf.github.io/protoconf/installation)

# Build

```sh
 docker build -t protoconf/protoconf-vizceral .
```

# Run

1. Run the `protoconf` agent in dev mode:

```sh
protoconf agent -dev .
```

2. Run the container:

```sh
docker run -p 18080:8080 protoconf/protoconf-vizceral -protoconf_addr=host.docker.internal:4300
```

3. Open your browser: http://localhost:18080

# Making changes

## Editing the code

1. Open `src/service/frontend.pinc`
2. Under `downstreams` comment out the `CHECKOUT_SERVICE`
3. run `protoconf compile .`

## Via Mutation

1. Run the mutation server

```sh
protoconf serve -post scripts/post.sh
```

2. Run a mutation command against the mutation server

```sh
protoconf mutate -path regions/count -proto google/protobuf/wrappers.proto -msg google.protobuf.UInt32Value -field value=5
```
