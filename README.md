# Loss Prevention Service

## Getting Started

### Dependencies
#### Hardware
- USB Webcam (Tested and validated using `Logitech C920`)

#### Software
- `git`
- `docker`
- `docker-compose`
- `rsp-sw-toolkit-gw`
- `inventory-suite`

### TL;DR
**_Want to just get up and running quickly?_**

Run this command:
```bash
make builder && make iterate
```

This command will create the docker builder image, compile the source code, builds application docker image, starts app in foreground

### Build Instructions
#### Pre-build
Build the docker image that is used to compile the source code. This should only have to be run once, or if `go.mod` is modified.
```bash
make builder
```

#### Compile and Run
Compile the Go source code
```bash
make build
```

Create the Docker image
```bash
make docker
```

Start the docker-compose containers in the foreground
```bash
make start
```

To stop the docker services gracefully, simply press `Ctrl-C` in your terminal. Press `Ctrl-C` a second time to kill the containers.

#### Makefile commands
Run the docker-compose containers in the background
```bash
make deploy
```

Tail the container logs
```bash
make tail
```

Stop or kill services running in the background
```bash
# shutdown containers gracefully
make down

# force kill containers
make kill
```

#### Makefile macro commands
These commands simulate running multiple commands for ease of use
```bash
# rebuild and start in the foreground
# equivalent to running: down, build, docker, start
make iterate      

# rebuild and start in the background
# equivalent to running: down, build, docker, deploy, tail
make iterate-d
```
