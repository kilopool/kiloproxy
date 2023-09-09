## Linux File Limit
If you want handle more than 1000 connections, you need to increase the limit of open files.

View current limit:
```bash
ulimit -n
```

Increase limit for current session:
```bash
ulimit -n 200000
```

To increase limit permanently add these lines to the end of `/etc/security/limits.conf`:
```
proxyuser hard nofile 200000
proxyuser soft nofile 200000
```

You may also need to increase the limit in `/proc/sys/fs/file-max`:
```bash
echo 400000 > /proc/sys/fs/file-max
```

## Adding swap
If your device has limited memory, Kilopool may benefit from adding swap.
You can follow this guide: https://www.digitalocean.com/community/tutorials/how-to-add-swap-space-on-ubuntu-22-04