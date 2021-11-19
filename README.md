# envoy-wasm-example

This is a simple example of using WASM Envoy filter. Envoy is acting as a forward proxy with a list of allowed domains taken from external API.

I know it's easy to do such thing without a custom filter, but the purpose was to learn how to use WASM filters in Envoy.

## Before you run it

Install `tinygo`: https://tinygo.org/getting-started/install/

Download Envoy and have it on PATH: https://www.envoyproxy.io/docs/envoy/latest/start/install#

## Build filter

```sh
tinygo build -o filter.wasm -scheduler=none -target=wasi ./main.go
```

## Prepare API

Instead of writing API that will serve a list of allowed domains, we can use https://mockapi.io/.

You need to create a 'project' there, add `domains` resource. Example URL: https://61975766af46280017e7e547.mockapi.io/domains

Data example returned from the API:

```json
[ 
  {
    "name": "google.pl"
  } 
]
```

Update envoy.yaml config with your mockapi endpoint host:

- in 'controlplane' cluster's `soccet_address`
- in the filter configuration

## Run it

```sh
envoy -c envoy.yaml -l info --concurrency 1
```

`--concurrency 1` to avoid mockapi.io rate limiting (by default envoy will run ~10 worker threads on my machine)

```sh
curl --proxy http://localhost:9000  https://google.com
```

`curl` request should succeed of fail based on the list from your API.

## Reference

- ["Extending Envoy Proxy - WASM Filter with Golang" blog post](https://medium.com/trendyol-tech/extending-envoy-proxy-wasm-filter-with-golang-9080017f28ea)
- ["Extending Envoy Using WebAssembly (Wasm)" YT video](https://www.youtube.com/watch?v=JFPJdNrcHSA)
- [Envoy WASM examples](https://github.com/tetratelabs/proxy-wasm-go-sdk/tree/main/examples)
