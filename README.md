# addr2me

[![Build and Push](https://github.com/k0st1an/addr2me/actions/workflows/image.yaml/badge.svg)](https://github.com/k0st1an/addr2me/actions/workflows/image.yaml)

- Web:
    - http://addr2.me
    - https://addr2.me
- DockerHub: https://hub.docker.com/r/k0st1an/addr2me
- GitHub: https://github.com/k0st1an/addr2me

## Example

```sh
curl addr2.me
79.143.107.6
```

## Usage
### Docker

```sh
docker pull k0st1an/addr2me
```
```sh
docker run --name addr2me --rm -p 7007:7007/tcp k0st1an/addr2me
```

### CLI

```sh
git clone https://github.com/k0st1an/addr2me.git && cd addr2me
```
```sh
make build
```
```sh
./addr2me -h
Usage of ./addr2me:
  -log-prefix string
    	log prefix (default "[addr2.me] ")
  -port string
    	port to listen on (default ":7007")
```
