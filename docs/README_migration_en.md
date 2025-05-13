## Migration Guide for Binary Version Deployed via run.sh
- **Back up old saves**
  - First, go to **Tools → Backup**, then click **Backup Now**.
  - Download the generated `.tgz` file.
  - Extract it locally and recompress it into a `.zip` file according to the import format.
- **Clear old data**
  - Open the server terminal.
  - Enter: `cd && rm -rf dmp* .klei DstMP.sdb`
- **Update the DMP to the latest version**
  - Run: `cd && echo 4 | ./run.sh`
- **Import old saves**
  - Refresh the webpage, register an account, and log in.
  - Create a new cluster.
  - Import the save file.

## Migration Guide for Docker
- **Back up old saves**
  - First, go to **Tools → Backup**, then click **Backup Now**.
  - Download the generated `.tgz` file.
  - Extract it locally and recompress it into a `.zip` file according to the import format.
- **Clear old data**
  - Delete Mapped Directories (e.g., config, .klei, etc.)
- **Update the DMP to the latest version**
  - Pull the Latest Docker Image and Restart
- **Import old saves**
  - Refresh the webpage, register an account, and log in.
  - Create a new cluster.
  - Import the save file.