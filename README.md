# mykmyk

Automate your blackbox pentest.
---
The output of the mykmyk is the report in HTML and files related to the scan.

Mykmyk supports:
- nmap
- ffuf
- httpx
- nuclei
- smb scan
- netcat
- sslscan
- rdp scan

# How to run
### 1.	Create folder for pentest:
```
mkdir <pentest-name>
```
### 2.	Create ***hosts*** file which contains targets for scan. 
Each IP/Domain from the new line:
```
vim hosts
```
Example:
```
pentest.co.uk
20.77.132.140
google.com
```
### 3.	Initialize mykmyk
this command creates config.yml. The config contains the configuration for each tool like nmap, ffuf, etc. This command creates a local config (in the current directory) and global under your home directory .config/mykmyk. Mykmyk will try to read the local config at first, if doesn't find then try to read the global one:
```
mykmyk init
```
### 4.	Run scan (run it in screen or tmux). 
Mykmyk asks for the username, domain name and credentials - which are used for **smb** and **rdp** scan:
```
mykmyk scan
```
### 5.	Check status:
```
mykmyk status -t hosts
```
### 6. Finish
When mykmyk is finished, zip the root folder for a particular pentest and copy it to your local machine. The folder contains the file ***mykmyk-output.html*** which is the report from the scan and files which contains output from the ran tool. In addition, the folder ***report-xml*** contains nmap scans in .xml format:
```
zip -r <pentest-name>.zip <pentest-name>
```
### 7. Find ports
```
mykmyk find -p 3389 -t hosts
```

# Additional information
Each task in the config file contains fields like ***active***, ***useCache***. You can switch off the task by changing active to false.
If your scan was aborted for some reason the mykmyk won't do a scan from zero but use already scanned information unless the filed useCache is false.
If you want to run for example ffuf scan once again but with a different wordlist you can do it in two ways:
1.	add an additional task to config - just copy the ffuf task, change the name and wordlist
2.	edit already created ffuf task by changing wordlist and setting useCache to false

# How to build
Run make from mykmyk/cmd/cli:
```
make build
```

# Configuration Documentation
## General Configuration
- **Output File**: `./mykmyk-output.html`
- **Workflow**: Consists of multiple tasks using different tools (e.g., `nmap`, `httpx`, `nuclei`, `ffuf`, etc.) to scan and analyze host targets from the input file.

## Workflow Tasks

### 1. `scope`
- **Type**: `filesystem`
- **Description**: Reads the list of hosts to be scanned.
- **Settings**:
  - `active`: `true` (task is enabled)
  - `useCache`: `true` (caching enabled)
- **Run Parameters**:
  - `input`: `./hosts` (input file containing the list of hosts)

### 2. `ST-scan`
- **Type**: `nmap`
- **Description**: Performs a TCP SYN scan on the specified hosts.
- **Source**: `scope`
- **Settings**:
  - `active`: `true`
  - `useCache`: `true`
  - `concurrency`: `3` (maximum number of concurrent scans)
- **Run Parameters**:
  - Arguments: `-sT -vvv -Pn --open --max-rate 1000 -p- -oA`

### 3. `SV-scan`
- **Type**: `nmap`
- **Description**: Conducts a service version scan.
- **Source**: `ST-scan`
- **Settings**:
  - `active`: `true`
  - `useCache`: `true`
  - `concurrency`: `4`
- **Run Parameters**:
  - Arguments: `-sV -sC -vvv -Pn -A --max-rate 1000 --version-all --open -oA`

### 4. `RMI-scan`
- **Type**: `nmap`
- **Description**: Scans for RMI services and dumps the registry.
- **Source**: `ST-scan`
- **Settings**:
  - `active`: `true`
  - `useCache`: `true`
  - `concurrency`: `3`
- **Run Parameters**:
  - Arguments: `-sT -vvv -Pn --script=+rmi-dumpregistry --open --max-rate 1000 -oA`

### 5. `httpx-scan`
- **Type**: `httpx`
- **Description**: Identifies http/https services on hosts.
- **Source**: `ST-scan`
- **Settings**:
  - `active`: `true`
  - `useCache`: `true`
  - `concurrency`: `3`
- **Run Parameters**:
  - Arguments: `-td -vhost`

### 6. `nuclei-scan`
- **Type**: `nuclei`
- **Description**: Runs vulnerability scans using predefined templates.
- **Source**: `httpx-scan`
- **Settings**:
  - `active`: `true`
  - `useCache`: `true`
  - `concurrency`: `3`
- **Run Parameters**:
  - Arguments: `-etags wordpress,dos,fuzz -ud ~/nuclei-templates -rl 3 -c 5 -ni -duc -nc`

### 7. `ffuf-scan`
- **Type**: `ffuf`
- **Description**: Performs directory brute-forcing on web applications.
- **Source**: `httpx-scan`
- **Settings**:
  - `active`: `true`
  - `useCache`: `true`
  - `concurrency`: `3`
- **Run Parameters**:
  - Arguments: `-w ~/wordlist/raft-large-directories.txt -ac -rate 300 -v`

### 8. `ssl-scan`
- **Type**: `sslscan`
- **Description**: Performs SSL analysis on hosts to identify SSL/TLS protocol configurations.
- **Source**: `httpx-scan`
- **Settings**:
  - `active`: `true`
  - `useCache`: `true`
  - `concurrency`: `3`
- **Run Parameters**:
  - Arguments: `--no-colour --iana-names`

### 9. `ssl-scan-psql`
- **Type**: `sslscan`
- **Description**: Performs SSL analysis specifically for PostgreSQL services.
- **Source**: `ST-scan`
- **Settings**:
  - `active`: `true`
  - `useCache`: `true`
  - `concurrency`: `3`
- **Run Parameters**:
  - Arguments: `--no-colour --iana-names --starttls-psql`

### 10. `nc-fingerprint`
- **Type**: `nc` (Netcat)
- **Description**: Runs a fingerprinting task using crafted commands.
- **Source**: `ST-scan`
- **Settings**:
  - `active`: `true`
  - `useCache`: `true`
  - `concurrency`: `5`
- **Run Parameters**:
  - Arguments: `<>()&;id aa()`

### 11. `smb-check`
- **Type**: `smb`
- **Description**: Checks credentials for SMB services and their configuration.
- **Source**: `ST-scan`
- **Settings**:
  - `active`: `true`
  - `useCache`: `true`
  - `concurrency`: `3`

### 12. `rdp-check`
- **Type**: `rdp`
- **Description**: Checks credentials for RDP services.
- **Source**: `ST-scan`
- **Settings**:
  - `active`: `true`
  - `useCache`: `true`
  - `port`: `3389`
  - `concurrency`: `3`

---

## Note
All tasks have caching enabled (`useCache: true`), ensuring that results are reused whenever possible.