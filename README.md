# send
A centralized deployment service and its corresponding CLI, written in Go.

## Requirements
- Go (latest version)

### Required variables:
Create a .envrc file in the repository by running the following and setting the correct values:
```bash
cp envrc.template .envrc
```

Using [`direnv`](https://direnv.net) is recommended. Otherwise, you need to source it using `source .env`.

## To run
After cloning, run
```
go build
```
then
```
./send
```

