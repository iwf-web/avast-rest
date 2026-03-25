# av-rest

Antivirus scanner with REST API. Docker image running Avast Business Antivirus for Linux with a lightweight REST interface.

Tech docs: https://repo.avcdn.net/linux-av/doc/avast-techdoc.pdf?inid=avastcom-linux-antivirus__avast-techdoc

## Prerequisites

An Avast Business Linux license is required. Either activate via code (once) or mount an existing license file.

https://www.avast.com/business/products/linux-antivirus#pc

### Generate a license file from an activation code

```bash
docker run --rm \
  -e AVAST_ACTIVATION_CODE=XXXX-XXXX-XXXX-XXXX \
  -v $(pwd)/license:/etc/avast \
  iwfwebsolutions/av-rest:latest
```

This writes `./license/license.avastlic` and exits. The activation code can only be redeemed a limited number of times.

## Run

```bash
docker run -p 9000:9000 \
  -v $(pwd)/license/license.avastlic:/etc/avast/license.avastlic \
  -v $(pwd)/scandata:/scandata \
  iwfwebsolutions/av-rest:latest
```


## Endpoints

| Endpoint                        | Method | Description                                                            |
|---------------------------------|--------|------------------------------------------------------------------------|
| `/scanFile?path=/absolute/path` | GET    | Scan a file by absolute path. Returns status, description and filename.|
| `/version`                      | GET    | Returns the Avast engine/VPS version.                                  |
| `/healthcheck`                  | GET    | Returns `200` if scanner is healthy and signatures are fresh.          |

### Status Codes

| Code | Meaning                                                              |
|------|----------------------------------------------------------------------|
| 200  | File is clean                                                        |
| 400  | Scanner returned a general error                                     |
| 406  | Threat found                                                         |
| 420  | Signatures are older than `HEALTHCHECK_MAX_SIGNATURE_AGE`            |
| 503  | Scanner is unreachable                                               |

### Example responses

**Clean file:**
```bash
$ curl "http://localhost:9000/scanFile?path=/tmp/report.pdf"
{"Status":"OK","Description":"","FileName":"/tmp/report.pdf"}
```

**Infected file:**
```bash
$ curl "http://localhost:9000/scanFile?path=/tmp/eicar.zip"
{"Status":"FOUND","Description":"EICAR Test-NOT virus!!!","FileName":"/tmp/eicar.zip"}
```

**Version:**
```bash
$ curl http://localhost:9000/version
{"Version":"26032406"}
```

**Healthcheck:**
```bash
$ curl -i http://localhost:9000/healthcheck
HTTP/1.1 200 OK
```

## Configuration

| Variable                        | Default           | Description                                                  |
|---------------------------------|-------------------|--------------------------------------------------------------|
| `PORT`                          | `9000`            | HTTP listening port                                          |
| `HEALTHCHECK_MAX_SIGNATURE_AGE` | `48`              | Max signature age in hours before `/healthcheck` returns 420 |
| `AVAST_TELEMETRY`               | `0`               | Avast telemetry reporting (0 = off, 1 = on)                  |
| `AVAST_STATISTICS`              | `0`               | Avast statistics reporting (0 = off, 1 = on)                 |
| `AVAST_COMMUNITY`               | `0`               | Avast community participation (0 = off, 1 = on)              |
| `AVAST_THREADS`                 | `0`               | Number of additional scanner threads (0 = only main thread)  |
| `AVAST_MAX_FILE_SIZE_TO_EXTRACT_MB` | `1000`        | Max file size in MB to extract from archives                 |
| `AVAST_MAX_COMPRESSION_RATIO`   | `100`             | Max compression ratio before treating archive as a bomb      |
| `AVAST_REPORT_PUP`              | `0`               | Report potentially unwanted programs (`-u` flag, 0/1)        |
| `AVAST_REPORT_TOOLS`            | `0`               | Report tools/hacktools (`-T` flag, 0/1)                      |

## Protocol Support

The service supports HTTP/1.1 and unencrypted HTTP/2 (h2c):

```bash
# HTTP/1.1
curl --http1.1 "http://localhost:9000/scanFile?path=/tmp/file.txt"

# H2C (unencrypted HTTP/2)
curl --http2-prior-knowledge "http://localhost:9000/scanFile?path=/tmp/file.txt"
```

## Development

### Build the Docker image

```bash
docker build -t iwfwebsolutions/av-rest:latest .
```
