# README / Install 

## Adjusting the Log Directory

In the provided `docker-compose.yml` file, the directory for the Arma Reforger server logs is set. By default, it's set to `/var/lib/pterodactyl/volumes/35b98dbb-5f3f-40b0-9d2c-2910153bc991/profile/logs`.

### Steps to Adjust:

1. **Adjust the Path:** Replace `/var/lib/pterodactyl/volumes/35b98dbb-5f3f-40b0-9d2c-2910153bc991/profile/logs` with the actual path where your Arma Reforger server logs are stored.

2. **Server Configuration:** Ensure that in your Arma Reforger server configuration, the `-logStats` option is set to `2000`. This ensures that the logs are captured correctly and in sufficient detail.

## Installing Docker and Docker Compose on Debian 12

### Install Docker:

1. **Download and run the Docker installation script:**
   ```bash
   curl -fsSL https://get.docker.com -o get-docker.sh
   sudo sh get-docker.sh
   ```

2. **Start Docker service:**
   ```bash
   sudo systemctl start docker
   sudo systemctl enable docker
   ```

### Install Docker Compose:

1. **Download Docker Compose:**
   ```bash
   sudo curl -L "https://github.com/docker/compose/releases/download/$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep -oP '"tag_name": "\K(.*)(?=")')/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
   ```

2. **Grant execution rights:**
   ```bash
   sudo chmod +x /usr/local/bin/docker-compose
   ```

3. **Verify the installation:**
   ```bash
   docker-compose --version
   ```

## Using the Docker-Compose File

1. **Navigate to the directory containing the `docker-compose.yml` file.**

2. **Start the services:**
   ```bash
   docker-compose up -d
   ```

   The `-d` flag ensures the containers run in the background.

3. **Check the status of the containers:**
   ```bash
   docker-compose ps
   ```

4. **Stop the containers:**
   ```bash
   docker-compose down
   ```

## Grafana Dashboard

To efficiently monitor the metrics of your Arma Reforger server, you can use the pre-configured Grafana dashboard available in the Git repository.

### Setting up the Grafana Dashboard:

1. **Start Grafana:** Open your Grafana interface, which runs on port `3000`.

2. **Add a data source:**
    - Navigate to **Configuration** (gear icon) > **Data Sources**.
    - Click on **Add data source**.
    - Select **Prometheus** as the data source type.
    - Enter the URL `http://prometheus:9090`.
    - Click **Save & Test** to verify the connection.

3. **Import the dashboard:**
    - Go to **Create** (plus icon) > **Import**.
    - Choose the option to upload a JSON file.
    - Upload the dashboard JSON file from the Git repository.
    - Select the previously added Prometheus data source when prompted.
    - Click **Import** to load the dashboard.

The dashboard provides a comprehensive overview of server performance, including real-time metrics and historical data.

## Troubleshooting and Support

Please be aware that there may still be bugs in the current configuration. If you encounter any issues, do not hesitate to open an issue in the Git repository. Describe the problem in as much detail as possible to facilitate a quick resolution.

Note: I will not be providing support via Discord. Please use the Git issue feature for communication and problem-solving.