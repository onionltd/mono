# HOWTO

## Build and deploy service image

```
$ cd services/gitsync
$ make docker/build
$ cd ../../ansible
$ ansible-playbook -i inventory.ini update_$stack.yaml -t gitsync
```
