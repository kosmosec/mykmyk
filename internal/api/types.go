package api

type TaskType string

const (
	FilesystemType TaskType = "filesystem"
	ExecType       TaskType = "exec"
	NmapType       TaskType = "nmap"
	HTTPXType      TaskType = "httpx"
	NucleiType     TaskType = "nuclei"
	FfufType       TaskType = "ffuf"
	Nc             TaskType = "nc"
	Smb            TaskType = "smb"
	SSLScan        TaskType = "sslscan"
	Rdp            TaskType = "rdp"
)
