# Rockset receiver webhook

A webhook receiver that runs as an AWS lambda and writes the payloads it receives to Rockset collections.

## Configuration

The webhook receiver is configured using environment variables. The following variables are required:

* `ROCKSET_APIKEY` - the Rockset API key to use.
* `ROCKSET_APISERVER` - the Rockset API server to use.
* `WORKSPACE` - the default Rockset workspace where collections are located.
* `PATHS` - a JSON object that maps webhook paths to Rockset collections.

The format of the `PATHS` variable that maps the webhook path to a Rockset collection:

```json
{
  "/github": {
    "collection": "github",
    "workspace": "webhooks"
  },
  "/pagerduty": {
    "collection": "pagerduty",
    "workspace": "webhooks"
  }
}
```

## Docker Image

The docker image is published at `docker.io/rockset/webhook`, and in the future we might publish a vanity URL
using [this](https://github.com/amancevice/terraform-aws-custom-ecr-domain) method.
