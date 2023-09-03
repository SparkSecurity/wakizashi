# Wakizashi Scrape Platform

## Credit

[Tencent Security Xuanwu Lab](https://xlab.tencent.com/en/)

## Architecture

Wakizashi has two components: manager and worker.

Manager is a web server that provides API for users to submit scraping tasks. 

Worker is a daemon that runs on different machines and executes scraping tasks.

## Deployment

### Manager

[All configurations](https://github.com/SparkSecurity/wakizashi/blob/main/manager/config/config.go)

1. Switch to `docker` folder
2. Replace `mq_pass` in `.env` with a random password, that will be used for worker to connect MQ.
3. Run `docker-compose up -d`
4. Run `./add-token.sh` to generate a token for API calls.

### Worker

[All configurations](https://github.com/SparkSecurity/wakizashi/blob/main/worker/config/config.go)

```bash
docker run --name wakizashi-worker -d --restart always -e MQ_URI="amqp://guest:<mq_pass>@<ip>:5672" ghcr.io/sparksecurity/wakizashi-worker:main
```

## API

Swagger Doc: \<manager endpoint\>/docs

## API (deprecated)

### Submit a task

```python3
import requests
import json

url = "http://<manager>:3033/task"

payload = json.dumps({
  "name": "example-task",
  "urls": [
    "https://api.ip.sb/ip",
    "https://google.com",
    "https://bing.com"
  ]
})

headers = {
  'Token': '<token>',
  'Content-Type': 'application/json'
}

response = requests.request("POST", url, headers=headers, data=payload)

taskID = response.json()["taskID"]
```

### Append URLs to a task

```python3
import requests
import json

url = "http://<manager>:3033/task/<task_id>"

payload = json.dumps({
  "urls": [
    "https://amazon.com"
  ]
})

headers = {
  'Token': '<token>',
  'Content-Type': 'application/json'
}

response = requests.request("PUT", url, headers=headers, data=payload)
```

### List scrape tasks

```python3
import requests

url = "http://<manager>:3033/task"

headers = {
  'Token': '<token>'
}

response = requests.request("GET", url, headers=headers)

/* response.json() = 
[
    {
        "id": "64781bc7635765e0dad9c405",
        "name": "test3",
        "userID": "64781bb81d62c61fb5d57d50",
        "createdAt": "2023-06-01T12:17:11.93+08:00"
    },
    {
        "id": "64787e4ece8d36ca7f903853",
        "name": "test-ip",
        "userID": "64781bb81d62c61fb5d57d50",
        "createdAt": "2023-06-01T19:17:34.413+08:00"
    }
]
*/
```

### Get statistics of a task

```python3
import requests

url = "http://<manager>:3033/task/<task_id>/statistics"

headers = {
  'Token': '<token>'
}

response = requests.request("GET", url, headers=headers)

/* response.json() =
{
    "total": 3,
    "successful": 1,
    "failed": 2,
    "inProgress": 0
}
*/
```

### Download results of a task

GET `http://<manager>:3033/task/<task_id>`

Response is a zip file, which structure is:

```
task_name.zip
├── data
│   ├── 44d45f44592c966e3049d15c6e2a50209d52168a55e82d2d31a058735304eea7
│   ├── 72179dada963ca9f154ea2844b614b40b3ba38c7dd99208aaef2e9fd58cca19e
│   ├── 7b6a484d04943fac714dadb783e8b0fb67fa1a94938507bde7a27b61682afd60
│   └── 84bc159725f637822a5fc08e6e6551cc7cc1ce11681e6913f10a88b7fae8eef9
└── index.json
```

`data` folder contains the body of each URL, named by its SHA256 hash.

`index.json` is a list of IDs, URLs and their corresponding body hash:

```json
[
  {
    "id": "64781bc7635765e0dad9c406",
    "url": "https://google.com",
    "bodyHash": "72179dada963ca9f154ea2844b614b40b3ba38c7dd99208aaef2e9fd58cca19e"
  },
  {
    "id": "64781e4167bb7a4c941a809d",
    "url": "https://codeforces.com",
    "bodyHash": "44d45f44592c966e3049d15c6e2a50209d52168a55e82d2d31a058735304eea7"
  },
  {
    "id": "64781e4167bb7a4c941a809e",
    "url": "https://atcoder.jp",
    "bodyHash": "84bc159725f637822a5fc08e6e6551cc7cc1ce11681e6913f10a88b7fae8eef9"
  },
  {
    "id": "64781f4d67bb7a4c941a80a0",
    "url": "https://codeforces.com/problemset/problem/1837/D",
    "bodyHash": "7b6a484d04943fac714dadb783e8b0fb67fa1a94938507bde7a27b61682afd60"
  }
]
```
