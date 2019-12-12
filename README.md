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

### Sensor Configuration
Login to the `RSP Controller` and set the `Personality` of a SINGLE sensor to `EXIT`. This is the sensor that will trigger recording events when a matching RFID tag moves near it.

#### Service Configuration
Modify the [`secrets/configuration.json`](secrets/configuration.json) with your camera and tag information

- `ipCameraStreamUrl` Stream URL for the IP Camera you wish to connect to. (Example: `"rtsp://user:pass@ipaddress:port"`)
- `epcFilter` Wildcard based filter of EPC tags to trigger on. (Example: `"3014*BEEF*"`)
- `skuFilter` Wildcard based filter of SKU/GTIN values to trigger on. (Example: `"123*78*"`)
- `emailSubscribers` String comma separated of emails to receive notifications. (Example: `"your@email.com,your@email2.com"`)

> **NOTE 1:** `skuFilter` and `epcFilter` must **BOTH** match for the tag to match. Typically you would set one or the other and then set the other field to match everything (`*`)

> **NOTE 2:** In regards to `skuFilter` and `epcFilter`, a value of `*` effectively matches every possible item. Also, the filter must match the whole EPC/SKU and not just a subset. For example, if the SKU value is `123456789`, a filter of `*345*`, `123*`, `*789`, `1*5*9` **WILL** match, however filters such as `1234`, `789`, `*8`, `12*56` will **NOT** match because they only match a *subset* of the SKU and not the whole value.

#### Build
Compile the Go source code, create the docker images, and start the docker swarm services

> :warning: **_Notice_**
>
> Replace `GIT_TOKEN=...` with your access token generated from `github.impcloud.net` like so: `GIT_TOKEN=abc34f2323fcda2ad23`

```bash
sudo GIT_TOKEN=... make iterate
```

> The first time you run this it may take quite some time. Grab some :coffee:.


### Web Interface
The web interface is integrated with the Angular [`Demo UI`](http://localhost:4200/loss-prevention). It provides a way to view previous recordings including any people/objects detected. Recordings can also be deleted from the web ui.

### Application Flow
- Make REST calls to the `EdgeX Command Service` to retrieve information about the RSP sensors. 
  - The application needs to know which RSP sensors are `EXIT` personality, as well as the aliases for each RSP in order to perform lookups of `alias -> device_id`.   
- Subscribe to `ZeroMQ` to receive `inventory_event` messages from EdgeX CoreData
  - When an event is received, pass it on the the Trigger Logic to determine whether to trigger a recording.

### Trigger Logic
> **ALL** Of the following conditions **MUST** be met for the recording to trigger
- Event type is `moved`
- Previous location is **not** an `EXIT` personality sensor
- Current location **is** an `EXIT` personality sensor
- SKU matches `skuFilter` wildcard from [`secrets/configuration.json`](secrets/configuration.json) (`"*"` matches everything)
- EPC matches `epcFilter` wildcard from [`secrets/configuration.json`](secrets/configuration.json) (`"*"` matches everything)
- Another recording is not currently in progress
