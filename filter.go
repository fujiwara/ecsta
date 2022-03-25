package ecsta

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/Songmu/prompter"
)

func (app *Ecsta) runFilter(src io.Reader, title string) (string, error) {
	command := app.config.FilterCommand
	if command == "" {
		return runInternalFilter(src, title)
	}
	var f *exec.Cmd
	if strings.Contains(command, " ") {
		f = exec.Command("sh", "-c", command)
	} else {
		f = exec.Command(command)
	}
	f.Stderr = os.Stderr
	p, _ := f.StdinPipe()
	go func() {
		io.Copy(p, src)
		p.Close()
	}()
	b, err := f.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute filter command: %w", err)
	}
	return string(bytes.TrimRight(b, "\r\n")), nil
}

func runInternalFilter(src io.Reader, title string) (string, error) {
	var items []string
	s := bufio.NewScanner(src)
	for s.Scan() {
		fmt.Println(s.Text())
		items = append(items, strings.Fields(s.Text())[0])
	}

	var input string
	for {
		input = prompter.Prompt("Enter "+title, "")
		if input == "" {
			continue
		}
		var found []string
		for _, item := range items {
			item := item
			if item == input {
				found = []string{item}
				break
			} else if strings.HasPrefix(item, input) {
				found = append(found, item)
			}
		}

		switch len(found) {
		case 0:
			fmt.Printf("no such item %s\n", input)
		case 1:
			fmt.Printf("%s=%s\n", title, found[0])
			return found[0], nil
		default:
			fmt.Printf("%s is ambiguous\n", input)
		}
	}
}
