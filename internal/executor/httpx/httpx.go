package httpx

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/kosmosec/mykmyk/internal/api"
	"github.com/kosmosec/mykmyk/internal/credsmanager"
	"github.com/kosmosec/mykmyk/internal/executor/abstract"
	"github.com/kosmosec/mykmyk/internal/model"
	"github.com/kosmosec/mykmyk/internal/sns"
	"github.com/kosmosec/mykmyk/internal/status"
	"gopkg.in/yaml.v2"
)

// TODO: wyciagnij wspolne rzeczy do jeden struktury. To wszystko sie wszedzie powtarza
type Httpx struct {
	Type          api.TaskType
	Name          string
	Source        string
	Concurrency   int
	waitForSignal chan bool
	signalDone    chan bool
	waitFor       string
	output        []model.Output
	mu            sync.Mutex
	once          sync.Once
	cache         cache
	isActive      bool
	isCacheActive bool
	args          []string
	sns           *sns.SNS
	consumer      chan model.Message
}

type scanResult struct {
	target string
	urls   []string
	err    error
}

func liveOutput(taskName string, target string, output []string) {
	fmt.Printf("Httpx scan %s, output for %s\n", taskName, target)
	for _, o := range output {
		fmt.Printf("%s\n", o)
	}
	fmt.Println()
}

func (h *Httpx) scanTarget(target string, msg model.Message, db *sql.DB) ([]string, error) {
	fmt.Printf("[+] Httpx scanning for %s started\n", target)
	cached, found := h.cache.get(target, h.Name)
	if h.isCacheActive && found {
		return cached, nil
	}

	err := status.AddTaskToStatus(db, h.Name, target, target)
	if err != nil {
		return nil, err
	}

	output, err := scan(target, msg.Ports, h.args)
	if err != nil {
		return nil, err
	}
	err = status.UpdateDoneTaskInStatus(db, h.Name, target, target)
	if err != nil {
		return nil, err
	}
	urls := getURLs(output)
	return urls, nil

}

func (h *Httpx) Run(ctx context.Context, in interface{}, db *sql.DB) error {
	task, err := h.unmarshal(in)
	if err != nil {
		return err
	}
	h.args = task.Args
	resultCh := make(chan scanResult, h.Concurrency)
	var wg sync.WaitGroup
	limiter := make(chan bool, h.Concurrency)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-h.consumer:
				if !ok {
					wg.Wait()
					close(resultCh)
					return
				}
				h.once.Do(func() {
					h.waitForTask()
				})
				limiter <- true
				wg.Add(1)
				// httpx gets one IP with multiple ports
				target := msg.Targets[0]
				go func(ctx context.Context, target string, msg model.Message) {
					defer func() {
						wg.Done()
						<-limiter
					}()
					for {
						select {
						case <-ctx.Done():
							return
						default:
							result, err := h.scanTarget(target, msg, db)
							resultCh <- scanResult{target: target, urls: result, err: err}
							return
						}
					}

				}(ctx, target, msg)
			}
		}

		// for msg := range h.consumer {
		// 	h.once.Do(func() {
		// 		h.waitForTask()
		// 	})
		// 	limiter <- true
		// 	wg.Add(1)
		// 	// httpx gets one IP with multiple ports
		// 	target := msg.Targets[0]
		// 	go func(target string, msg model.Message) {
		// 		defer func() {
		// 			wg.Done()
		// 			<-limiter
		// 		}()
		// 		result, err := h.scanTarget(target, msg)
		// 		resultCh <- scanResult{target: target, urls: result, err: err}
		// 	}(target, msg)
		// }
		// wg.Wait()
		// close(resultCh)
	}()

	for r := range resultCh {
		if r.err != nil {
			return err
		}
		liveOutput(h.Name, r.target, r.urls)
		h.sendMessageToSNS(r.target, r.urls)
		h.saveToFile(r.target, r.urls)
		h.collectOutput(r.target, r.urls)
	}

	h.sns.CloseTopic(h.Name)
	h.signalDoneTask()
	return nil
}

func (h *Httpx) collectOutput(target string, urls []string) {
	output := model.Output{
		Type:    h.Type,
		Name:    h.Name,
		Target:  target,
		Results: model.Results{Data: urls},
	}
	h.mu.Lock()
	h.output = append(h.output, output)
	h.mu.Unlock()
}

func (h *Httpx) sendMessageToSNS(host string, urls []string) error {
	if len(urls) != 0 {
		h.sns.SendMessage(h.Name, model.Message{Targets: urls})
	}
	return nil
}

func (h *Httpx) saveToFile(target string, urls []string) error {
	path := fmt.Sprintf("./%s/%s", target, h.Name)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, u := range urls {
		if _, err = f.WriteString(u + "\n"); err != nil {
			return err
		}
	}
	return nil
}

func (h *Httpx) New(task api.Task, sns *sns.SNS, creds credsmanager.Credentials) abstract.Executor {
	return &Httpx{
		Type:          task.Type,
		Name:          task.Name,
		Source:        task.Source,
		Concurrency:   task.Concurrency,
		waitFor:       task.WaitFor,
		isActive:      task.Active,
		isCacheActive: task.UseCache,
		output:        make([]model.Output, 0),
		cache:         cache{},
		sns:           sns,
	}
}

func cutTechInfo(liveUrls []string) []string {
	urlsWithoutTech := make([]string, 0)
	for _, u := range liveUrls {
		urlWithoutTech := strings.Split(u, " ")[0]
		if urlWithoutTech != "" {
			urlsWithoutTech = append(urlsWithoutTech, strings.TrimSpace(urlWithoutTech))
		}
	}
	return urlsWithoutTech
}

func getURLs(output string) []string {
	urls := strings.Split(output, "\n")
	urlList := make([]string, 0)
	for _, t := range urls {
		if strings.TrimSpace(t) != "" {
			urlList = append(urlList, strings.TrimSpace(t))
		}
	}
	urlList = cutTechInfo(urlList)
	return urlList
}

func (h *Httpx) GetConcurrency() int {
	return h.Concurrency
}

func (h *Httpx) signalDoneTask() {
	if h.signalDone != nil {
		h.signalDone <- true
	}
}

func (h *Httpx) waitForTask() {
	if h.waitForSignal != nil {
		<-h.waitForSignal
	}
}

func (h *Httpx) SetDoneSignal(signalDone chan bool) {
	h.signalDone = signalDone
}

func (h *Httpx) SetWaitForSignal(waitForSignal chan bool) {
	h.waitForSignal = waitForSignal
}

func (h *Httpx) GetWaitFor() string {
	return h.waitFor
}

func (h *Httpx) GetWaitForSignal() chan bool {
	return h.waitForSignal
}

func (h *Httpx) GetDoneSignal() chan bool {
	return h.signalDone
}

func (h *Httpx) Output() []model.Output {
	return h.output
}

func (h *Httpx) GetType() api.TaskType {
	return h.Type
}

func (h *Httpx) GetSource() string {
	return h.Source
}
func (h *Httpx) HasSource() bool {
	if h.Source != "" {
		return true
	}
	return false
}

func (h *Httpx) GetName() string {
	return h.Name
}

func (h *Httpx) unmarshal(in interface{}) (*Task, error) {
	var task Task
	raw, err := yaml.Marshal(in)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(raw, &task); err != nil {
		return nil, err
	}
	return &task, err
}

func (h *Httpx) SetConsumer(c chan model.Message) {
	h.consumer = c
}

func (h *Httpx) IsActive() bool {
	return h.isActive
}
