# send
A centralized deployment service and its corresponding CLI, written in Go.

## Requirements
- [Go (latest version)](https://golang.org/)
- [Python 3.6 or above](https://www.python.org/downloads/)
- [virtualenv](https://virtualenv.pypa.io/en/stable/)
- [Vagrant with Virtualbox](https://www.vagrantup.com/downloads.html)
- [Ansible](http://docs.ansible.com/ansible/latest/installation_guide/intro_installation.html)

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

