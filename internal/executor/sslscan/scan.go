package sslscan

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"

	"github.com/kosmosec/mykmyk/internal/binary"
	"github.com/kosmosec/mykmyk/internal/status"
)

func scan(host string, urls []string, args []string, db *sql.DB, taskName string) (string, []string, error) {
	if _, err := os.Stat(host); os.IsNotExist(err) {
		os.Mkdir(host, 0775)
	}

	var urlsToCheck string
	pathsToReport := make([]string, 0)
	for _, u := range urls {
		actualArgs := make([]string, 0)
		actualArgs = append(actualArgs, args...)
		parsedUrl, _ := url.Parse(u)
		sslReportName := fmt.Sprintf("./%s/sslReport-%s-%s.xml", host, parsedUrl.Hostname(), parsedUrl.Port())
		sslReportArg := fmt.Sprintf("--xml=%s", sslReportName)
		actualArgs = append(actualArgs, sslReportArg)
		actualArgs = append(actualArgs, u)
		output, _, err := binary.Run("sslscan", actualArgs, nil)
		if err != nil {
			return "", nil, err
		}
		err = status.UpdateDoneTaskInStatus(db, taskName, host, u)
		if err != nil {
			return "", nil, err
		}
		pathsToReport = append(pathsToReport, sslReportName)
		urlsToCheck += output.String()
	}

	return urlsToCheck, pathsToReport, nil
}
