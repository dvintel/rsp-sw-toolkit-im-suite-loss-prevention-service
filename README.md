# Loss Prevention Service

## Getting Started

### Dependencies
#### Hardware
One of the following:
- USB Webcam (Tested and validated using `Logitech C920`)
- PoE Camera
- WiFi Camera

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


### Web Interface
The web interface is integrated with the Angular [`Demo UI`](http://localhost:4200/loss-prevention). It provides a way to view previous recordings including any people/objects detected. Recordings can also be deleted from the web ui.

### Application Flow
- Make REST calls to the `EdgeX Command Service` to retrieve information about the RSP sensors. 
  - The application needs to know which RSP sensors are `EXIT` personality, as well as the aliases for each RSP in order to perform lookups of `alias -> device_id`.   
- Subscribe to `ZeroMQ` to receive `inventory_event` messages from EdgeX CoreData
  - When an event is received, pass it on the the Trigger Logic to determine whether to trigger a recording.

### Trigger Logic
- Event type is `moved`
- Previous location is **not** an `EXIT` personality sensor
- Current location **is** an `EXIT` personality sensor
- SKU matches `skuFilter` wildcard from [`docker-compose.yml`](docker-compose.yml) (`"*"` matches everything)
- EPC matches `epcFilter` wildcard from [`docker-compose.yml`](docker-compose.yml) (`"*"` matches everything)
- Another recording is not currently in progress
