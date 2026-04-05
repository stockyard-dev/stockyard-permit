# Stockyard Permit

**Self-hosted permit and license tracking**

Part of the [Stockyard](https://stockyard.dev) family of self-hosted tools.

## Quick Start

```bash
curl -fsSL https://stockyard.dev/tools/permit/install.sh | sh
```

Or with Docker:

```bash
docker run -p 9811:9811 -v permit_data:/data ghcr.io/stockyard-dev/stockyard-permit
```

Open `http://localhost:9811` in your browser.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `9811` | HTTP port |
| `DATA_DIR` | `./permit-data` | SQLite database directory |
| `STOCKYARD_LICENSE_KEY` | *(empty)* | License key for unlimited use |

## Free vs Pro

| | Free | Pro |
|-|------|-----|
| Limits | 5 records | Unlimited |
| Price | Free | Included in bundle or $29.99/mo individual |

Get a license at [stockyard.dev](https://stockyard.dev).

## License

Apache 2.0

