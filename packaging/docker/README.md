# Docker compose

The docker-compose provides an app container for Aisc.
To prepare your machine to run docker compose execute
```
mkdir -p aisc && cd aisc
wget -q https://raw.githubusercontent.com/ethersphere/aisc/master/packaging/docker/docker-compose.yml
wget -q https://raw.githubusercontent.com/ethersphere/aisc/master/packaging/docker/env -O .env
```
Set all configuration variables inside `.env`

If you want to run node in full mode, set `Aisc_FULL_NODE=true`

Aisc requires an Ethereum endpoint to function. Obtain a free Infura account and set:
- `Aisc_BLOCKCHAIN_RPC_ENDPOINT=wss://sepolia.infura.io/ws/v3/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`

Set aisc password by either setting `Aisc_PASSWORD` or `Aisc_PASSWORD_FILE`

If you want to use password file set it to
- `Aisc_PASSWORD_FILE=/password`

Mount password file local file system by adding
```
- ./password:/password
```
to aisc volumes inside `docker-compose.yml`

Start it with
```
docker-compose up -d
```

From logs find URL line with `on sepolia you can get both sepolia eth and sepolia aisc from` and prefund your node
```
docker-compose logs -f aisc-1
```

Update services with
```
docker-compose pull && docker-compose up -d
```

## Running multiple Aisc nodes
It is easy to run multiple aisc nodes with docker compose by adding more services to `docker-compose.yaml`
To do so, open `docker-compose.yaml`, copy lines 4-54 and past this after line 54 (whole aisc-1 section).
In the copied lines, replace all occurrences of `aisc-1` with `aisc-2` and adjust the `API_ADDR` and `P2P_ADDR` and `DEBUG_API_ADDR` to respectively `1733`, `1734` and `127.0.0.1:1735`
Lastly, add your newly configured services under `volumes` (last lines), such that it looks like:
```yaml
volumes:
  aisc-1:
  aisc-2:
```

If you want to create more than two nodes, simply repeat the process above, ensuring that you keep unique name for your aisc services and update the ports
