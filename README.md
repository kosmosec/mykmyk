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

# How to run
1.	Create folder for pentest:
```
mkdir <pentest-name>
```
2.	Create ***hosts*** file which contains targets for scan. Each IP/Domain from the new line:
```
vim hosts
```
Example:
```
pentest.co.uk
20.77.132.140
google.com
```
3.	Initialize mykmyk - this command creates config.yml. The config contains the configuration for each tool like nmap, ffuf, etc. This command creates a local config (in the current directory) and global under your home directory .config/mykmyk. Mykmyk will try to read the local config at first, if doesn't find then try to read the global one:
```
mykmyk init
```
4.	Run scan (run it in screen or tmux). Mykmyk asks for the username, domain name and credentials - which are used for smb scan:
```
mykmyk scan
```
5.	Check status:
```
mykmyk status -t hosts
```
10.	When mykmyk is finished, zip the root folder for a particular pentest and copy it to your local machine. The folder contains the file ***mykmyk-output.html*** which is the report from the scan and files which contains output from the ran tool. In addition, the folder ***report-xml*** contains nmap scans in .xml format:
```
zip -r <pentest-name>.zip <pentest-name>
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