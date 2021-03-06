package file

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sync"

	"github.com/fatih/color"
	"github.com/nomad-software/grope/cli"
)

const MAX_NUMBER_OF_WORKERS = 100

type WorkerQueue struct {
	Closed chan bool
	Group  *sync.WaitGroup
	Input  chan UnitOfWork
	Output *cli.Output
}

type UnitOfWork struct {
	File    string
	Pattern *regexp.Regexp
}

func (this *WorkerQueue) Start() {
	go this.Output.Start()

	life := make(chan bool)

	for i := 0; i <= MAX_NUMBER_OF_WORKERS; i++ {
		go this.worker(life)
	}

	for i := 0; i <= MAX_NUMBER_OF_WORKERS; i++ {
		<-life
	}

	close(this.Output.Console)
	<-this.Output.Closed

	this.Closed <- true
}

func (this *WorkerQueue) Close() {
	close(this.Input)
	<-this.Closed
}

func (this *WorkerQueue) worker(death chan<- bool) {
	for work := range this.Input {

		file, err := os.Open(work.File)
		if err != nil {
			fmt.Fprintln(cli.Stderr, color.RedString(err.Error()))
			this.Group.Done()
			continue
		}

		lines := make([]cli.Line, 0)
		var lineNumber int

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lineNumber++
			if work.Pattern.MatchString(scanner.Text()) {
				lines = append(lines, cli.Line{
					Number: lineNumber,
					Line:   work.Pattern.ReplaceAllString(scanner.Text(), color.New(color.FgHiRed, color.Bold).SprintFunc()("$0")),
				})
			}
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(cli.Stderr, color.RedString(fmt.Sprintf("Error scanning %s - %s", work.File, err.Error())))
			this.Group.Done()
			continue
		}

		if len(lines) > 0 {
			this.Output.Console <- cli.Match{
				File:  work.File,
				Lines: lines,
			}
		}

		file.Close()
		this.Group.Done()
	}

	death <- true
}
