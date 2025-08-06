# SystemD Configuration for XRP

This directory contains systemd service configuration for running XRP as a system service.

## Installation

1. **Create XRP user and group:**
   ```bash
   sudo useradd --system --no-create-home --shell /bin/false xrp
   ```

2. **Install XRP binary:**
   ```bash
   sudo cp xrp /usr/local/bin/
   sudo chown root:root /usr/local/bin/xrp
   sudo chmod 755 /usr/local/bin/xrp
   ```

3. **Create configuration directory:**
   ```bash
   sudo mkdir -p /etc/xrp
   sudo mkdir -p /opt/xrp/plugins
   sudo mkdir -p /var/log/xrp
   sudo chown xrp:xrp /var/log/xrp
   ```

4. **Install configuration file:**
   ```bash
   sudo cp examples/config.example.json /etc/xrp/config.json
   sudo chown root:xrp /etc/xrp/config.json
   sudo chmod 640 /etc/xrp/config.json
   ```

5. **Install plugins (if using):**
   ```bash
   sudo cp examples/plugins/*.so /opt/xrp/plugins/
   sudo chown root:xrp /opt/xrp/plugins/*.so
   sudo chmod 644 /opt/xrp/plugins/*.so
   ```

6. **Install systemd service:**
   ```bash
   sudo cp xrp.service /etc/systemd/system/
   sudo systemctl daemon-reload
   ```

## Usage

```bash
# Enable and start the service
sudo systemctl enable xrp
sudo systemctl start xrp

# Check status
sudo systemctl status xrp

# View logs
sudo journalctl -u xrp -f

# Reload configuration (sends SIGHUP)
sudo systemctl reload xrp

# Stop the service
sudo systemctl stop xrp
```

## Configuration

Edit `/etc/xrp/config.json` to configure:
- Upstream server settings
- Redis cache configuration
- Plugin configuration
- MIME type handling

The service will automatically reload configuration when you run `systemctl reload xrp`.

## Security Features

The systemd service includes comprehensive security hardening:

- **User isolation**: Runs as dedicated `xrp` user with minimal privileges
- **Filesystem protection**: Read-only access to system directories
- **Network restrictions**: Limited to required address families
- **Process restrictions**: No new privileges, restricted syscalls
- **Memory protection**: Write+execute memory protection (disabled for Go plugins)

## Logs

Service logs are written to:
- **systemd journal**: `journalctl -u xrp`
- **Application logs**: `/var/log/xrp/` (if configured in config.json)

## Dependencies

The service requires:
- **Redis**: Configured as a dependency (`Requires=redis.service`)
- **Network**: Waits for network connectivity before starting

## Troubleshooting

Common issues:

1. **Plugin loading failures**: Check file permissions and paths in `/opt/xrp/plugins/`
2. **Redis connection errors**: Ensure Redis service is running and accessible
3. **Permission denied**: Verify xrp user has access to required directories
4. **Configuration errors**: Check syntax with `xrp -config /etc/xrp/config.json -check`