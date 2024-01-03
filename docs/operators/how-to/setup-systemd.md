# Set up a systemd service

This page will take you through how to set up a `systemd` service for go-waku.

`systemd` is used in order to have a command or a program run when your device boots (i.e. add it as a service).
Once this is done, you can start/stop enable/disable from the linux prompt.

!!! abstract "`systemd`"
    [`systemd`](https://systemd.io/) is a service manager designed specifically for Linux: it cannot be used on Windows / Mac.
    You can find out more about `systemd` [here](https://fedoramagazine.org/what-is-an-init-system/).

!!! note "Package manager installations"
    When installing go-waku via your package manager, a user and service will already have been created for you and you can skip straight to the configuration section.

### 1. Create the service file

`systemd` services are created by placing a [service](https://www.freedesktop.org/software/systemd/man/latest/systemd.service.html) file in `/etc/systemd/system`, or, if go-waku was installed by a package manager, `/usr/lib/systemd/system`.

A good starting point is the [example service file](https://raw.githubusercontent.com/waku-org/go-waku/scripts/linux/waku.service) in the go-waku repository.

```sh
# Download example service file and save it to `/etc/systemd/system/go-waku.service`
curl -s https://raw.githubusercontent.com/waku-org/go-waku/scripts/linux/go-waku.service | sudo tee /etc/systemd/system/waku.service > /dev/null
```

The format of service files is documented in the [systemd manual](https://www.freedesktop.org/software/systemd/man/latest/systemd.service.html).

!!! note
    go-waku has two return codes for errors:
    - `1` returned for recoverable errors
    - `166` returned for non-recoverable errors.
    The example service file uses 166 in `RestartPreventExitStatus` to prevent automated restarts for non recoverable errors.

### 2. Configure your service

Service is configured by editing the service file directly, or using `systemctl edit` to create an override.

```sh
# Edit the systemd file to match your installation
sudo vi /etc/systemd/system/waku.service

# If you installed go-waku via the package manager, use `systemctl edit` instead
sudo systemctl edit waku.service
```

!!! note
    The example assumes go-waku was installed in `/usr/bin/waku`.
    If you installed go-waku elsewhere, make sure to update this path.

### 3. Notify systemd of the newly added service

Every time you add or update a service, the `systemd` daemon must be notified of the changes:

```sh
sudo systemctl daemon-reload
```

### 4. Start the service

```sh
# start go-waku node
sudo systemctl start waku

# (Optional) Set go-waku to start automatically at boot
sudo systemctl enable waku
```

### 5. Check the status of the service

`systemctl status` will show if go-waku is up and running, or has stopped for some reason.

```sh
sudo systemctl status waku.service
```

You can also follow the logs using the following command:

```sh
sudo journalctl -uf waku.service
```

This will show you the waku logs at the default setting. Press `ctrl-c` to stop following the logs.

To rewind logs — by one day, say — run:

```sh
sudo journalctl -u waku.service --since yesterday
```
