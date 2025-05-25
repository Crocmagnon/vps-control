# vps-control

```shell
cp .mise.toml.dist .mise.toml
```

Fill the `.mise.toml` file with OVH API credentials from [this page](https://www.ovh.com/auth/api/createToken?GET=/vps&GET=/vps/*&GET=/vps/*/task/*&POST=/vps/*/reboot).

```shell
mise task -l
mise run run
```

To deploy a new version:
```shell
mise deploy
```
