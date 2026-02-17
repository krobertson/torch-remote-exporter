# Space Engineers Torch Exporter

This is a simple implementation of a Prometheus Exporter for Space Engineers
server running [Torch server](https://torchapi.com) and using the
[Torch Remote](https://github.com/PveTeam/TorchRemote/) plugin to get API
functionality.

The Space Engineers RCON functionality seems fairly unreliable, so was
diffciult to export metrics from it for instrumentation.

## Usage

Pull the Docker image and run it. Currently it is hardcoded to port 9090. It
requires the following to be set as environment variables:

* `TORCH_HOST` - The hostname/IP of the Torch server.
* `TORCH_PORT` - The port of the Torch Remote API.
* `TORCH_PASS` - The security key from the `TorchRemote.cfg` file.
