package status

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/kosmosec/mykmyk/internal/api"
)

func Status(ctx context.Context, targetFile string, cfg api.Config) error {
	db, err := sql.Open("sqlite3", "status.db")
	if err != nil {
		return err
	}

	defer db.Close()

	targets, err := loadScope(targetFile)
	if err != nil {
		return err
	}
	statuses := make(map[string][]task)
	for _, t := range targets {
		rows, err := db.Query("SELECT * FROM Status WHERE Target = ?", t)
		if err != nil {
			return err
		}
		defer rows.Close()
		taskForTarget := make([]task, 0)
		for rows.Next() {
			var id int
			var target string
			var taskName string
			var taskTarget string
			var done bool

			err := rows.Scan(&id, &target, &taskName, &taskTarget, &done)
			if err != nil {
				return err
			}
			taskForTarget = append(taskForTarget, task{
				taskName:   taskName,
				taskTarget: taskTarget,
				done:       done,
			})

		}
		statuses[t] = taskForTarget
	}
	for target, tasks := range statuses {
		fmt.Printf("Status for target %s\n", target)
		tasksPerTaskName := make(map[string][]task)
		for _, t := range tasks {
			tasksPerTaskName[t.taskName] = append(tasksPerTaskName[t.taskName], t)
		}

		for tName, result := range tasksPerTaskName {
			var taskInProcess int
			var taskDone int
			for _, r := range result {
				if r.done {
					taskDone++
				} else {
					taskInProcess++
				}
			}
			fmt.Printf("%20s %20d/%d\n", tName, taskDone, taskDone+taskInProcess)
		}
		fmt.Println()
	}
	return nil
}

type task struct {
	taskName   string
	taskTarget string
	done       bool
}

func loadScope(filePath string) ([]string, error) {
	rawScope, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read scope from file %v: %w", filePath, err)
	}
	targets := strings.Split(string(rawScope), "\n")
	targetList := make([]string, 0)
	for _, t := range targets {
		if t != "" {
			targetList = append(targetList, strings.TrimSpace(t))
		}
	}
	return targetList, nil
}

func AddTaskToStatus(db *sql.DB, taskName string, target string, taskTarget string) error {
	stmt, err := db.Prepare("INSERT INTO Status (Target, TaskName, TaskTarget, Done) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(target, taskName, taskTarget, false)
	if err != nil {
		return err
	}
	return nil
}

func UpdateDoneTaskInStatus(db *sql.DB, taskName string, target string, taskTarget string) error {
	stmt, err := db.Prepare("UPDATE Status SET Done = ? WHERE TaskName = ? AND Target = ? AND TaskTarget = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(true, taskName, target, taskTarget)
	if err != nil {
		return err
	}
	return nil
}
