# addr2me

Repo: https://github.com/k0st1an/addr2me


## Usage
### Docker

```sh
docker pull k0st1an/addr2me
```
```sh
docker run --name addr2me --rm -p 7007:7007/tcp k0st1an/addr2me
```

Or

```sh
make docker-run
```

### CLI

```sh
$ git clone https://github.com/k0st1an/addr2me.git
```
```sh
$ make build
```
```sh
./addr2me -h
Usage of ./addr2me:
  -log-prefix string
    	log prefix (default "[addr2.me] ")
  -port string
    	port to listen on (default ":7007")
```
