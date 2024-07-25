package executor

import (
	"github.com/kosmosec/mykmyk/internal/api"
	"github.com/kosmosec/mykmyk/internal/credsmanager"
	"github.com/kosmosec/mykmyk/internal/executor/abstract"
	"github.com/kosmosec/mykmyk/internal/executor/exec"
	"github.com/kosmosec/mykmyk/internal/executor/ffuf"
	"github.com/kosmosec/mykmyk/internal/executor/filesystem"
	"github.com/kosmosec/mykmyk/internal/executor/httpx"
	"github.com/kosmosec/mykmyk/internal/executor/nc"
	"github.com/kosmosec/mykmyk/internal/executor/nmap"
	"github.com/kosmosec/mykmyk/internal/executor/nuclei"
	"github.com/kosmosec/mykmyk/internal/executor/smb"
	"github.com/kosmosec/mykmyk/internal/executor/sslscan"
	"github.com/kosmosec/mykmyk/internal/sns"
)

type Creator interface {
	New(task api.Task, sns *sns.SNS, creds credsmanager.Credentials) abstract.Executor
}

// Really needed Creator. Maybe Executor will be good
var Registered = map[api.TaskType]Creator{
	api.ExecType:       &exec.Exec{},
	api.FilesystemType: &filesystem.Filesystem{},
	api.NmapType:       &nmap.Nmap{},
	api.HTTPXType:      &httpx.Httpx{},
	api.NucleiType:     &nuclei.Nuclei{},
	api.FfufType:       &ffuf.Ffuf{},
	api.Nc:             &nc.Nc{},
	api.Smb:            &smb.Smb{},
	api.SSLScan:        &sslscan.SSLScan{},
}
