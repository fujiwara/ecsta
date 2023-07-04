package ecsta

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/Songmu/prompter"
)

func (app *Ecsta) runFilter(ctx context.Context, src io.Reader, title string) (string, error) {
	command := app.Config.FilterCommand
	if command == "" {
		return runInternalFilter(ctx, src, title)
	}
	var f *exec.Cmd
	if strings.Contains(command, " ") {
		f = exec.CommandContext(ctx, "sh", "-c", command)
	} else {
		f = exec.CommandContext(ctx, command)
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

func runInternalFilter(ctx context.Context, src io.Reader, title string) (string, error) {
	var items []string
	s := bufio.NewScanner(src)
	for s.Scan() {
		fmt.Println(s.Text())
		items = append(items, strings.Fields(s.Text())[0])
	}

	result := make(chan string)
	go func() {
		var input string
		for {
			input = prompter.Prompt("Enter "+title, "")
			if input == "" {
				select {
				case <-ctx.Done():
					return
				default:
				}
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
				result <- found[0]
				return
			default:
				fmt.Printf("%s is ambiguous\n", input)
			}
		}
	}()
	select {
	case <-ctx.Done():
		return "", ErrAborted
	case r := <-result:
		return r, nil
	}
}
