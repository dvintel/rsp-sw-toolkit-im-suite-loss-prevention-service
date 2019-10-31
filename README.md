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
Compile the Go source code, create the docker images, and start the docker-compose services

> :warning: **_Notice_**
>
> Replace `GIT_TOKEN=...` with your access token generated from `github.impcloud.net` like so: `GIT_TOKEN=abc34f2323fcda2ad23`

```bash
sudo GIT_TOKEN=... make -j iterate
```

> The first time you run this it may take quite some time. Grab some :coffee:.

> To stop the docker services gracefully, simply press `Ctrl-C` in your terminal. Press `Ctrl-C` a second time to kill the containers.
