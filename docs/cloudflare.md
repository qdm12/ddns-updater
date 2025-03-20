# DDNS-Updater Setup Guide for Ubuntu Server

This guide provides step-by-step instructions for setting up DDNS-Updater by qdm12 on an Ubuntu Server with Docker. This solution will automatically update your Cloudflare DNS record when your public IP changes and provide a monitoring dashboard.

## Prerequisites

- Ubuntu Server with Docker installed
- Sudo/root access to the server
- Cloudflare account with API token
- Domain managed by Cloudflare (e.g., cloudcommand.org)

## Step 1: Create a Cloudflare API Token

1. Log in to your Cloudflare dashboard at https://dash.cloudflare.com
2. Go to "My Profile" > "API Tokens" > "Create Token"
3. Select "Create Custom Token"
4. Name it "DDNS Updater"
5. Under "Permissions":
   - Zone - DNS - Edit
   - Zone - Zone - Read
6. Under "Zone Resources":
   - Include - Specific zone - your domain (e.g., cloudcommand.org)
7. **Important**: Set "TTL" to "No expiration" (do not select a date range) or your token will expire and break your DDNS
8. Click "Continue to summary" then "Create Token"
9. Copy the generated token (you'll only see it once)

## Step 2: Prepare Your Docker Environment

```bash
# Create a directory for DDNS-Updater
mkdir -p ~/ddns-updater
cd ~/ddns-updater
```

This directory (`~/ddns-updater`) will be mounted into the container and will store your configuration file and update history. The important paths are:

- Host path: `~/ddns-updater/config.json` (expands to `/home/yourusername/ddns-updater/config.json`)
- Container path: `/updater/data/config.json`

## Step 3: Start the DDNS-Updater (Confirmed Working Method)

Based on our extensive testing, this is the configuration that definitely works:

### Using a config.json file (CONFIRMED WORKING)

We have confirmed this method works with both the latest version and v2.5.0:

```bash
# Create directory for configuration
mkdir -p ~/ddns-updater

# Create the config.json file with your API token
cat > ~/ddns-updater/config.json << 'EOF'
{
  "settings": [
    {
      "provider": "cloudflare",
      "zone_identifier": "YOUR_ZONE_ID",
      "domain": "cloudcommand.org",
      "host": "rdgateway02",
      "token": "YOUR_API_TOKEN",
      "ttl": 120
    }
  ]
}
EOF

# Run the container with mounted volume
docker run -d \
  --name ddns-updater \
  --restart always \
  -p 8000:8000/tcp \
  -v ~/ddns-updater:/updater/data \
  -e LOG_LEVEL=info \
  qmcgaw/ddns-updater:latest
```

### Alternative Method (NOT TESTED)

While the container documentation suggests environment variables can also be used, we have NOT confirmed if this method works reliably with our specific setup:

```bash
docker run -d \
  --name ddns-updater \
  --restart always \
  -p 8000:8000/tcp \
  -e SETTINGS_1_PROVIDER=cloudflare \
  -e SETTINGS_1_ZONE_IDENTIFIER=YOUR_ZONE_ID \
  -e SETTINGS_1_DOMAIN=cloudcommand.org \
  -e SETTINGS_1_HOST=rdgateway02 \
  -e SETTINGS_1_TOKEN=YOUR_API_TOKEN \
  -e SETTINGS_1_TTL=120 \
  -e PERIOD=5m \
  -e LOG_LEVEL=info \
  qmcgaw/ddns-updater:latest
```

**Important Notes:**
- Only the config.json file method has been thoroughly tested and confirmed working in our environment
- Replace `YOUR_ZONE_ID` with your actual Cloudflare Zone ID (looks like: b5b434545550d4af9e402c2d01516274)
- Replace `YOUR_API_TOKEN` with your actual Cloudflare API token
- Splitting the domain into `domain` (cloudcommand.org) and `host` (rdgateway02) is critical for proper functioning
- The `TTL` parameter is required (120 seconds is a good value for dynamic DNS)
- The mounted volume approach ensures configuration persistence across container restarts

## Step 4: Check Container Status

```bash
# Check if the container is running properly
docker ps | grep ddns-updater

# View initial logs
docker logs ddns-updater
```

## Step 5: Access the Web Dashboard

The web dashboard will be available at `http://YOUR_UBUNTU_SERVER_IP:8000`

You can check this from your server to make sure it's responding:

```bash
# Test if the web UI is accessible locally
curl http://localhost:8000
```

## Step 6: Configure Automatic Updates for the Container

```bash
# Install Watchtower to automatically update DDNS-Updater container
docker run -d \
  --name watchtower \
  --restart always \
  -v /var/run/docker.sock:/var/run/docker.sock \
  containrrr/watchtower \
  --cleanup \
  --interval 86400 \
  ddns-updater
```

This will automatically update the DDNS-Updater container when new versions are available.

## Step 7: Test and Verify

After a few minutes, the web UI should show your domain with the current IP address and "Up to date" status. You can also verify it worked by:

1. Checking the Cloudflare DNS dashboard
2. Using DNS lookup tools:

```bash
# Install dig if needed
apt-get update && apt-get install -y dnsutils

# Check your record
dig rdgateway02.cloudcommand.org
```

## Troubleshooting

### Common Errors and Solutions

#### "Missing X-Auth-Key, X-Auth-Email or Authorization headers" Error (Code 9106)
- This is a Cloudflare authentication error that commonly occurs after ISP modem restarts
- **Quick fix**: Restart the container with `docker restart ddns-updater`
- If restart doesn't work, recreate the container using the recommended config.json approach in Step 3
- This error indicates one of these issues:
  1. Using an incorrect authentication method (Global API Key instead of API Token)
  2. The API Token has expired or been revoked
  3. The domain and host are not correctly split in your configuration
  4. The configuration wasn't properly applied after a network interruption
- **Solution**: 
  ```bash
  # First check logs for detailed error information
  docker logs ddns-updater
  
  # Recreate the container using the config.json method (most reliable)
  docker rm -f ddns-updater
  
  # Create a proper config.json file with split domain/host
  mkdir -p ~/ddns-updater
  cat > ~/ddns-updater/config.json << 'EOF'
  {
    "settings": [
      {
        "provider": "cloudflare",
        "zone_identifier": "YOUR_ZONE_ID",
        "domain": "cloudcommand.org",
        "host": "rdgateway02",
        "token": "YOUR_API_TOKEN",
        "ttl": 120
      }
    ]
  }
  EOF
  
  # Run with the mounted config.json
  docker run -d \
    --name ddns-updater \
    --restart always \
    -p 8000:8000/tcp \
    -v ~/ddns-updater:/updater/data \
    -e LOG_LEVEL=debug \
    qmcgaw/ddns-updater:latest
  ```
- If you're seeing this error repeatedly after ISP modem restarts, consider creating a simple script to automatically restart the container when your internet connection is restored

#### "zone identifier is not set" Error
- Make sure you're using the correct Zone ID from Cloudflare
- The parameter should be named `zone_identifier` (not `zone_id`)

#### "TTL is not set" Error
- Add the `ttl` parameter to your configuration (e.g., `"ttl":120`)

#### "Cannot unmarshal object into Go struct field" Error
- This occurs when using incorrect JSON structure
- Use the redundant approach shown in Step 3

#### "Found no setting to update record" Error
- This means the configuration isn't being properly loaded
- The redundant approach in Step 3 helps resolve this issue

#### Web UI shows empty table
- Check logs for errors: `docker logs ddns-updater`
- Verify your API token has the correct permissions
- Try restarting the container: `docker restart ddns-updater`

### Viewing Detailed Logs

```bash
# View all logs
docker logs ddns-updater

# View last 100 log entries
docker logs --tail 100 ddns-updater

# Follow logs in real-time
docker logs -f ddns-updater
```

### Enabling Debug Mode

For more detailed logs:

```bash
docker rm -f ddns-updater
docker run -d \
  --name ddns-updater \
  --restart always \
  -p 8000:8000/tcp \
  -e SETTINGS_1_PROVIDER=cloudflare \
  -e SETTINGS_1_ZONE_IDENTIFIER=YOUR_ZONE_ID \
  -e SETTINGS_1_DOMAIN=rdgateway02.cloudcommand.org \
  -e SETTINGS_1_API_TOKEN=YOUR_API_TOKEN \
  -e SETTINGS_1_TTL=120 \
  -e PERIOD=5m \
  -e LOG_LEVEL=debug \
  -e CONFIG='{"settings":[{"provider":"cloudflare","zone_identifier":"YOUR_ZONE_ID","domain":"rdgateway02.cloudcommand.org","api_token":"YOUR_API_TOKEN","ttl":120}]}' \
  qmcgaw/ddns-updater
```

## Troubleshooting Cloudflare Authentication Errors

If you encounter authentication errors with Cloudflare after changing your IP (such as when restarting an ISP modem), try these solutions:

### Solution for "Missing X-Auth-Key, X-Auth-Email or Authorization headers" or "DNS name is invalid" Errors

The most reliable fix for Cloudflare authentication issues is to:

1. Use API Token authentication (not Global API Key)
2. Split the domain and subdomain correctly in your configuration
3. Mount a persistent volume for the config.json file
4. Set appropriate TTL values

Here's the complete working solution (works with both latest and v2.5.0):

```bash
# Create directory if it doesn't exist
mkdir -p ~/ddns-updater

# Create the config.json file with your API token
cat > ~/ddns-updater/config.json << 'EOF'
{
  "settings": [
    {
      "provider": "cloudflare",
      "zone_identifier": "b5b434545550d4af9e402c2d01516274",
      "domain": "cloudcommand.org",
      "host": "rdgateway02",
      "token": "YOUR_API_TOKEN",
      "ttl": 120
    }
  ]
}
EOF

# Stop and remove existing container
docker stop ddns-updater
docker rm ddns-updater

# Run the container with latest version
docker run -d \
  --name ddns-updater \
  --restart always \
  -p 8000:8000/tcp \
  -v ~/ddns-updater:/updater/data \
  -e LOG_LEVEL=debug \
  qmcgaw/ddns-updater:latest
```

**Key Points:**
- This configuration works with both the latest version and v2.5.0
- Split domain into two parts:
  - `domain`: Your root domain (e.g., cloudcommand.org)
  - `host`: Your subdomain prefix (e.g., rdgateway02)
- Mount a volume to preserve your configuration
- Use API Token authentication instead of Global API Key
- Setting LOG_LEVEL to debug helps with troubleshooting

This configuration ensures the DDNS-Updater will continue working even after ISP modem restarts and IP changes.

### File Locations

The configuration file is stored in these locations:

- **Host Machine**: `~/ddns-updater/config.json` (expands to `/home/yourusername/ddns-updater/config.json`)
- **Inside Container**: `/updater/data/config.json`

These locations are linked through the Docker volume mapping: `-v ~/ddns-updater:/updater/data`

To edit the configuration:
```bash
# Edit config with nano
nano ~/ddns-updater/config.json

# After editing, restart the container to apply changes
docker restart ddns-updater
```

### Version Compatibility Notes

- Both the latest version and v2.5.0 work correctly with the configuration shown above
- If you encounter issues with the latest version, you can try v2.5.0 which has been extensively tested:
  ```bash
  docker run -d \
    --name ddns-updater \
    --restart always \
    -p 8000:8000/tcp \
    -v ~/ddns-updater:/updater/data \
    qmcgaw/ddns-updater:v2.5.0
  ```

### Debugging Tips

If authentication issues persist:

1. Check container logs with `docker logs ddns-updater` for specific error messages
2. Try using the `-e LOG_LEVEL=debug` option to get more detailed logs
3. Verify that your Cloudflare API token is still valid by testing it with [Cloudflare's API documentation](https://api.cloudflare.com/)
4. Try recreating the API token in Cloudflare's dashboard

Remember that Cloudflare occasionally makes API changes that might require updates to your configuration.

## Conclusion and Key Learnings

Through extensive testing, we've definitively confirmed:

1. **The config.json file approach with mounted volume is VERIFIED WORKING**
2. **Both the latest container version and v2.5.0 work correctly** with the proper configuration
3. **Critical elements for success:**
   - Using API Token authentication (not Global API Key)
   - Splitting the domain into separate "domain" and "host" fields
   - Mounting a persistent volume for the config.json file
   - Setting appropriate TTL values

The web interface provides a convenient way to monitor your DDNS update status, and the detailed logs help with troubleshooting if any issues occur.

We have NOT confirmed whether the environment variables approach works reliably with our setup, so we recommend using the config.json method that has been proven to work.

## Why We Moved from OPNsense
We moved away from OPNsense's built-in DDNS client due to persistent reliability issues, including "list index out of range" errors, failed updates, and inconsistent behavior. The qdm12/ddns-updater solution provides far greater reliability, better error handling, and a user-friendly web interface to monitor status.

## Common Commands for Management

```bash
# Restart the container
docker restart ddns-updater

# Stop the container
docker stop ddns-updater

# Start the container
docker start ddns-updater

# Remove container and recreate it
docker rm -f ddns-updater
# Then run the docker run command from Step 3 again
```

This solution will maintain your Cloudflare DNS record with your current public IP address and provide a user-friendly dashboard to monitor updates and status. It's more reliable than OPNsense's built-in DDNS client and resolves the "list index out of range" error that can occur with ddclient.

## References

### Official Documentation
- [DDNS-Updater GitHub Repository](https://github.com/qdm12/ddns-updater) - Official project repository with comprehensive documentation
- [DDNS-Updater Cloudflare Documentation](https://github.com/qdm12/ddns-updater/blob/master/docs/cloudflare.md) - Specific guide for Cloudflare configuration
- [Docker Hub - qmcgaw/ddns-updater](https://hub.docker.com/r/qmcgaw/ddns-updater) - Official Docker image
- [DDNS-Updater Configuration Options](https://github.com/qdm12/ddns-updater/blob/master/docs/configuration.md) - All available configuration options

### Cloudflare Resources
- [Cloudflare API Documentation](https://developers.cloudflare.com/api/tokens/) - Documentation for Cloudflare API tokens
- [Cloudflare DNS Documentation](https://developers.cloudflare.com/dns/) - General DNS configuration in Cloudflare
- [Cloudflare Zone ID Location](https://developers.cloudflare.com/fundamentals/setup/find-account-and-zone-ids/) - How to find your Zone ID

### Docker Resources
- [Docker Run Reference](https://docs.docker.com/engine/reference/run/) - Documentation for docker run command options
- [Watchtower GitHub Repository](https://github.com/containrrr/watchtower) - For automatic container updates
- [Docker Environment Variables](https://docs.docker.com/engine/reference/commandline/run/#env) - How environment variables work in Docker

### Troubleshooting Resources
- [Common Docker Issues](https://docs.docker.com/engine/reference/commandline/logs/) - How to view logs and troubleshoot containers
- [Cloudflare Community Forum](https://community.cloudflare.com/) - Community support for Cloudflare issues
- [DDNS-Updater Issues Page](https://github.com/qdm12/ddns-updater/issues) - Known issues and community solutions
