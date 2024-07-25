package ffuf

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"

	"github.com/kosmosec/mykmyk/internal/binary"
	"github.com/kosmosec/mykmyk/internal/status"
)

func scan(host string, urls []string, args []string, prefix string, db *sql.DB, taskName string) (string, []string, error) {
	if _, err := os.Stat(host); os.IsNotExist(err) {
		os.Mkdir(host, 0775)
	}

	var fuzzedURLs string
	pathsToReport := make([]string, 0)
	for _, u := range urls {
		urlToFUZZ := prepareURL(u, prefix)
		actualArgs := make([]string, 0)
		actualArgs = append(actualArgs, args...)
		actualArgs = append(actualArgs, "-u", urlToFUZZ)
		actualArgs = append(actualArgs, "-of", "html")
		parsedUrl, _ := url.Parse(u)
		ffufReportName := fmt.Sprintf("./%s/ffufReport-%s-%s-%s.html", host, taskName, parsedUrl.Hostname(), parsedUrl.Port())
		actualArgs = append(actualArgs, "-o", ffufReportName)
		output, _, err := binary.Run("ffuf", actualArgs, nil)
		if err != nil {
			return "", nil, err
		}
		err = status.UpdateDoneTaskInStatus(db, taskName, host, u)
		if err != nil {
			return "", nil, err
		}
		pathsToReport = append(pathsToReport, ffufReportName)
		fuzzedURLs += output.String()
	}

	return fuzzedURLs, pathsToReport, nil
}

func prepareURL(url string, prefix string) string {
	var urlWithFUZZ string
	if prefix != "" {
		urlWithFUZZ = fmt.Sprintf("%s%s/%s", url, prefix, "FUZZ")
		return urlWithFUZZ
	}
	urlWithFUZZ = fmt.Sprintf("%s/%s", url, "FUZZ")
	return urlWithFUZZ
}
