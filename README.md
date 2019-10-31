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

### Build Instructions
#### Pre-build
This command only needs to be run once and will take a long time.
```bash
sudo GIT_TOKEN=xxx make prepare
```

#### Compile and Run
Compile the Go source code and builder docker container
```bash
sudo make build
```

Start the docker-compose containers in the foreground
```bash
make start
```

To stop the docker services gracefully, simply press `Ctrl-C` in your terminal. Press `Ctrl-C` a second time to kill the containers.
