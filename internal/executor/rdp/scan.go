package rdp

import (
	"fmt"
	"os"
	"time"

	"github.com/kosmosec/mykmyk/internal/credsmanager"
)

const TC_RDP = 1

func scan(host string, port int, args []string, creds credsmanager.Credentials) string {
	if _, err := os.Stat(host); os.IsNotExist(err) {
		os.Mkdir(host, 0775)
	}

	output := checkUserCredentials(host, port, creds)
	return output

}

func checkUserCredentials(hostname string, port int, creds credsmanager.Credentials) string {
	fmt.Printf("[+] RDP login for %s started\n", hostname)
	var result string
	err := RdpConn(hostname, creds.Domain, creds.User, creds.Password, port, 3*time.Second)
	if err != nil {
		result = fmt.Sprintf("[-] Unsuccessful (%v/%v) RDP login %v %v; %v\n", hostname, port, creds.User, creds.Password, err)
		return result
		//return "", errors.Errorf("[-] (%v/%v) rdp %v %v %v", hostname, port, creds.User, creds.Password, err)
	}

	if creds.Domain != "" {
		result = fmt.Sprintf("[+] Successful RDP login %v:%v:%v\\%v %v\n", hostname, port, creds.Domain, creds.User, creds.Password)
	} else {
		result = fmt.Sprintf("[+] Successful RDP login %v:%v:%v %v\n", hostname, port, creds.User, creds.Password)
	}
	return result
}
