package scanner

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	_ "github.com/mattn/go-sqlite3"

	"github.com/pkg/errors"

	"github.com/kosmosec/mykmyk/internal/api"
	"github.com/kosmosec/mykmyk/internal/credsmanager"
	"github.com/kosmosec/mykmyk/internal/executor"
	"github.com/kosmosec/mykmyk/internal/executor/abstract"
	"github.com/kosmosec/mykmyk/internal/model"
	"github.com/kosmosec/mykmyk/internal/sns"
)

func Scan(ctx context.Context, cfg api.Config) error {

	tasks := make([]abstract.Executor, 0)
	tasksRunMap := make(map[string]api.Task)

	sns := sns.New()
	creds := credsmanager.New()
	creds.RequestForCredentials()

	sigs := make(chan os.Signal, 1)
	cleanup := make(chan bool)
	signal.Notify(sigs, syscall.SIGINT)
	go func() {
		<-sigs
		fmt.Println("[!] Aborted!")
		<-cleanup
		os.Exit(1)
	}()

	for _, task := range cfg.Workflow.Tasks {
		concreteExecutor, found := executor.Registered[task.Type]
		if !found {
			return errors.Errorf("unsupported task type %s", task.Type)
		}

		executor := concreteExecutor.New(task, &sns, *creds)
		tasks = append(tasks, executor)
		tasksRunMap[task.Name] = task
	}

	db, err := createStatusDatabase()
	if err != nil {
		return err
	}

	defer db.Close()

	createTopics(tasks, &sns)
	err = addConsumersToTopics(tasks, &sns)
	if err != nil {
		return err
	}
	createWaitForConnection(tasks)

	var wg sync.WaitGroup
	go func() {
		for {
			if err := ctx.Err(); err != nil {
				fmt.Println(err)
				prettyOutput(cfg.OutputFile, tasks)
				cleanup <- true
				return
			}
		}
	}()
	for _, t := range tasks {
		wg.Add(1)
		go func(e abstract.Executor) {
			defer wg.Done()
			if e.IsActive() {
				err := e.Run(ctx, tasksRunMap[e.GetName()].Run, db)
				if err != nil {
					log.Fatalf("scanner: %s for task %s", err, e.GetName())
				}
			}
		}(t)
	}
	wg.Wait()

	prettyOutput(cfg.OutputFile, tasks)

	if _, err := os.Stat("./report-xml"); os.IsNotExist(err) {
		err := os.Mkdir("./report-xml", 0755)
		if err != nil {
			log.Fatalf("create report folder: %s", err)
		}
	}

	nmapScanName := firstNmapTaskName(cfg.Workflow.Tasks)
	if nmapScanName == "" {
		log.Fatalf("unable to find nmap task name for report")
	}
	nmapScans := findFile(".", nmapScanName+".xml")
	for i, ns := range nmapScans {
		bytesRead, err := os.ReadFile("./" + ns)

		if err != nil {
			log.Fatalf("try to find %s %s", ns, err)
		}

		dest := fmt.Sprintf("%s-%d.xml", "./report-xml/"+nmapScanName, i)
		err = os.WriteFile(dest, bytesRead, 0644)

		if err != nil {
			log.Fatalf("try to write copy content of %s %s", ns, err)
		}
	}

	return nil
}

func prettyOutput(reportName string, tasks []abstract.Executor) {
	scanResults := make(map[string][]model.Output)
	for _, t := range tasks {
		output := t.Output()
		for _, o := range output {
			scanResults[o.Target] = append(scanResults[o.Target], o)
		}
	}
	f, err := os.Create(reportName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	tmpl, err := template.New("").Parse(htmlReport)
	if err != nil {
		log.Fatal(err)
	}
	err = tmpl.Execute(f, scanResults)
	if err != nil {
		log.Fatal(err)
	}
}

func createTopics(tasks []abstract.Executor, sns *sns.SNS) {
	for i := range tasks {
		if tasks[i].IsActive() {
			sns.CreateTopic(tasks[i].GetName())
		}
	}
}

func firstNmapTaskName(tasks []api.Task) string {
	for _, t := range tasks {
		if t.Type == "nmap" {
			return t.Name
		}
	}
	return ""
}

func addConsumersToTopics(tasks []abstract.Executor, sns *sns.SNS) error {
	for i := range tasks {
		if tasks[i].HasSource() && tasks[i].IsActive() {

			if !isSourceOfTaskIsActive(tasks[i], tasks) {
				return errors.Errorf("The task %s has a inactive source %s", tasks[i].GetName(), tasks[i].GetSource())
			}
			concurrency := tasks[i].GetConcurrency()
			if concurrency == 0 {
				concurrency = 1
			}
			consumer := make(chan model.Message, concurrency)
			sns.AddConsumer(tasks[i].GetSource(), tasks[i].GetName(), consumer)
			tasks[i].SetConsumer(consumer)
		}
	}
	return nil
}

func isSourceOfTaskIsActive(currentTask abstract.Executor, tasks []abstract.Executor) bool {
	for _, t := range tasks {
		if currentTask.GetSource() == t.GetName() {
			if !t.IsActive() {
				return false
			}
		}
	}
	return true
}

func createWaitForConnection(tasks []abstract.Executor) {
	for i := range tasks {
		for j := range tasks {
			if tasks[j].GetName() == tasks[i].GetWaitFor() {
				// buffered because we do not know which gorutine runs first
				signal := make(chan bool, 1)
				tasks[i].SetWaitForSignal(signal)
				tasks[j].SetDoneSignal(signal)
			}
		}
	}
}

// return folder/fileName.ext
func findFile(root string, fileName string) []string {
	a := make([]string, 0)
	filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Name() == fileName {
			a = append(a, s)
		}
		return nil
	})
	return a
}

func createStatusDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "status.db")
	if err != nil {
		return nil, err
	}

	dbSchema := `
	DROP TABLE IF EXISTS Status;
	CREATE TABLE Status (ID INTEGER PRIMARY KEY AUTOINCREMENT, Target TEXT, TaskName TEXT, TaskTarget, Done BOOLEAN);
	`
	_, err = db.Exec(dbSchema)
	if err != nil {
		return nil, err
	}
	return db, nil
}

var htmlReport string = `
<!DOCTYPE html>
<html>
<head>
	<title>Report from scan</title>
	<style type="text/css">
	
body { background: #dedede; font-family: 'Droid sans', Helvetica, Arial, sans-serif; color: #404042; -webkit-font-smoothing: antialiased; }
#container { width: 930px; padding: 0 15px; margin: 20px auto; background-color: #ffffff; }
table { font-family: Arial, sans-serif; }
a:link, a:visited { color: #ff6633; text-decoration: none; }
a:hover, a:active { color: #e24920; text-decoration: underline; }
h1 { font-size: 1.6em; line-height: 1.4em; font-weight: normal; color: #404042; }
h2 { font-size: 1.3em; line-height: 1.2em; padding: 0; margin: 0.8em 0 0.3em 0; font-weight: normal; color: #404042;}
h4 { font-size: 1.0em; line-height: 1.2em; padding: 0; margin: 0.8em 0 0.3em 0; font-weight: bold; color: #404042;}
.rule { height: 0px; border-top: 1px solid #404042; padding: 0; margin: 20px -15px 0 -15px; }
.title { color: #ffffff; background: #1e517e; margin: 0 -15px 10px -15px; overflow: hidden; }
.title h1 { color: #ffffff; padding: 10px 15px; margin: 0; font-size: 1.8em; }
.title img { float: right; display: inline; padding: 1px; }
.heading { background: #404042; margin: 0 -15px 10px -15px; padding: 0; display: inline-block; overflow: hidden; }
.heading img { float: right; display: inline; margin: 8px 10px 0 10px; padding: 0; }
.code { font-family: 'Courier New', Courier, monospace; }
table.overview_table { border: 2px solid #e6e6e6; margin: 0; padding: 5px;}
table.overview_table td.info { padding: 5px; background: #dedede; text-align: right; border-top: 2px solid #ffffff; border-right: 2px solid #ffffff; }
table.overview_table td.info_end { padding: 5px; background: #dedede; text-align: right; border-top: 2px solid #ffffff; }
table.overview_table td.colour_holder { padding: 0px; border-top: 2px solid #ffffff; border-right: 2px solid #ffffff; }
table.overview_table td.colour_holder_end { padding: 0px; border-top: 2px solid #ffffff; }
table.overview_table td.label { padding: 5px; font-weight: bold; }
table.summary_table td { padding: 5px; background: #dedede; text-align: left; border-top: 2px solid #ffffff; border-right: 2px solid #ffffff; }
table.summary_table td.icon { background: #404042; }
.colour_block { padding: 5px; text-align: right; display: block; font-weight: bold; }
.high_certain { border: 2px solid #f32a4c; color: #ffffff; background: #f32a4c; }
.high_firm { border: 2px solid #f997a7; background: #f997a7; }
.high_tentative { border: 2px solid #fddadf; background: #fddadf; }
.medium_certain { border: 2px solid #ff6633; color: #ffffff; background: #ff6633; }
.medium_firm { border: 2px solid #ffb299; background: #ffb299; }
.medium_tentative { border: 2px solid #ffd9cc; background: #ffd9cc; }
.low_certain { border: 2px solid #0094ff; color: #ffffff; background: #0094ff; }
.low_firm { border: 2px solid #7fc9ff; background: #7fc9ff; }
.low_tentative { border: 2px solid #bfe4ff; background: #bfe4ff; }
.info_certain { border: 2px solid #7e8993; color: #ffffff; background: #7e8993; }
.info_firm { border: 2px solid #b9ced2; background: #b9ced2; }
.info_tentative { border: 2px solid #dae9ef; background: #dae9ef; }
.row_total { border: 1px solid #dedede; background: #fff; }
.grad_mark { padding: 4px; border-left: 1px solid #404042; display: inline-block; }
.bar { margin-top: 3px; }
.TOCH0 { font-size: 1.0em; font-weight: bold; word-wrap: break-word; }
.TOCH1 { font-size: 0.8em; text-indent: -20px; padding-left: 50px; margin: 0; word-wrap: break-word; }
.TOCH2 { font-size: 0.8em; text-indent: -20px; padding-left: 70px; margin: 0; word-wrap: break-word; }
.BODH0 { font-size: 1.6em; line-height: 1.2em; font-weight: normal; padding: 10px 15px; margin: 0 -15px 10px -15px; display: inline-block; color: #ffffff; background-color: #1e517e; width: 100%; word-wrap: break-word; }
.BODH0 a:link, .BODH0 a:visited, .BODH0 a:hover, .BODH0 a:active { color: #ffffff; text-decoration: none; }
.BODH1 { font-size: 1.3em; line-height: 1.2em; font-weight: normal; padding: 13px 15px; margin: 0 -15px 0 -15px; display: inline-block; width: 100%; word-wrap: break-word; }
.BODH1 a:link, .BODH1 a:visited, .BODH1 a:hover, .BODH1 a:active { color: #404042; text-decoration: none; }
.BODH2 { font-size: 1.0em; font-weight: bold; line-height: 2.0em; width: 100%; word-wrap: break-word; }
.PREVNEXT { font-size: 0.7em; font-weight: bold; color: #ffffff; padding: 3px 10px; border-radius: 10px;}
.PREVNEXT:link, .PREVNEXT:visited { color: #ff6633 !important; background: #ffffff !important; border: 1px solid #ff6633 !important; text-decoration: none; }
.PREVNEXT:hover, .PREVNEXT:active { color: #fff !important; background: #e24920 !important; border: 1px solid #e24920 !important; text-decoration: none; }
.TEXT { font-size: 0.8em; padding: 0; margin: 0; word-wrap: break-word; }
TD { font-size: 0.8em; }
.HIGHLIGHT { background-color: #fcf446; }
.rr_div { border: 2px solid #1e517e; width: 916px; word-wrap: break-word; -ms-word-wrap: break-word; margin: 0.8em 0; padding: 5px; font-size: 0.8em; max-height: 300px; overflow-y: auto; }

#table-of-content {
	position: fixed;
	top: 100px; /* Adjust this value to position the table of content */
	right: 0;
	width: 200px;
	background-color: #f8f8f8;
	border: 1px solid #ddd;
	padding: 10px;
  }
  
  #table-of-content ul {
	list-style: none;
	padding: 0;
	margin: 0;
  }
  
  #table-of-content li {
	margin-bottom: 5px;
  }
  
  #table-of-content a {
	text-decoration: none;
	color: #333;
  }
  
  #table-of-content a:hover {
	color: #000;
  }

</style>


</head>
<body>
	<div id="table-of-content">
	  <h2>Table of Contents</h2>
	  <ul>
		{{ range $target, $outputs := .}}
	    <li><a href="#{{ $target }}" tabindex="1">{{ $target }}</a></li>
		{{ end }}
	  </ul>
	</div>

	<div id="container">
	<div class="title">
		<h1>Mykmyk scanner report</h1>
	</div>
	{{range $target, $outputs := .}}
		<span class="BODH0" id="{{ $target }}">{{ $target }}</span>
		{{ range $output := $outputs}}
			<span class="TEXT">
			
			<h2>{{ $output.Type }}</h2>
			<h3>{{ $output.Name }}</h3>
			</span>
			
			
			{{if eq $output.Type "httpx" }}
				<div class="rr_div">
				{{ range $d := $output.Results.Data}}
					<p><a href="{{ $d }}">{{ $d }}</a></p>
				{{ end }}
				</div>
			{{ else if eq $output.Type "ffuf" }}
				<p>Ffuf reports in html</p>
				{{ range $ffufReportPath := $output.Results.ReportPaths }}
					<p><a href="{{ $ffufReportPath }}">{{ $ffufReportPath }}</a></p>
				{{ end }}
			{{ else }}
				<div class="rr_div">
					{{ range $d := $output.Results.Data }}
						<p><span>{{ $d }}</span></p>
					{{ end }}
				</div>
			{{ end }}
			<div class="rule"></div>
		{{end}}
	{{end}}
	</div>
</body>
</html>
`
