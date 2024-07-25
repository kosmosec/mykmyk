package nc

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/kosmosec/mykmyk/internal/status"
)

func scan(host string, ports []string, args []string, taskName string, db *sql.DB) (string, error) {
	if _, err := os.Stat(host); os.IsNotExist(err) {
		os.Mkdir(host, 0775)
	}
	var fingeredServices string

	for _, p := range ports {
		service, err := fingerprint(host, p, args)
		if err != nil {
			return "", err
		}
		err = status.UpdateDoneTaskInStatus(db, taskName, host, fmt.Sprintf("%s:%s", host, p))
		if err != nil {
			return "", err
		}
		output := fmt.Sprintf("%s %s\n %s\n\n", host, p, service)
		fingeredServices += output

	}
	return fingeredServices, nil
}

func fingerprint(host string, port string, args []string) (string, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%s", host, port), 5*time.Second)
	if err != nil {
		return "", err
	}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	var wg sync.WaitGroup
	buf := make([]byte, 1024)
	outputBuf := make([]byte, 0)
	for _, a := range args {
		wg.Add(1)
		go func(a string) {
			defer wg.Done()
			_, _ = conn.Write([]byte(a))
			// _, _ = conn.Write([]byte("<>()&;id"))
			// _, _ = conn.Write([]byte("aa()"))
		}(a)
	}
	for {
		var err error
		_, err = conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Println("nc", err)
			}
			break
		}
		outputBuf = append(outputBuf, buf...)
	}
	conn.Close()
	wg.Wait()

	return string(outputBuf), nil
}
