package ecsta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/olekukonko/tablewriter"
)

func newTaskFormatter(w io.Writer, t string, hasHeader bool) (taskFormatter, error) {
	switch t {
	case "table":
		return newTaskFormatterTable(w, hasHeader), nil
	case "tsv":
		return newTaskFormatterTSV(w, hasHeader), nil
	case "json":
		return newTaskFormatterJSON(w), nil
	default:
		return nil, fmt.Errorf("unknown task formatter: %s", t)
	}
}

type taskFormatter interface {
	AddTask(types.Task)
	Close()
}

var taskFormatterColumns = []string{
	"ID",
	"TaskDefinition",
	"Instance",
	"LastStatus",
	"DesiredStatus",
	"CreatedAt",
	"Group",
	"Type",
}

func taskToColumns(task types.Task) []string {
	return []string{
		arnToName(*task.TaskArn),
		arnToName(*task.TaskDefinitionArn),
		arnToName(aws.ToString(task.ContainerInstanceArn)),
		aws.ToString(task.LastStatus),
		aws.ToString(task.DesiredStatus),
		task.CreatedAt.In(time.Local).Format(time.RFC3339),
		aws.ToString(task.Group),
		string(task.LaunchType),
	}
}

type taskFormatterTable struct {
	table *tablewriter.Table
}

func newTaskFormatterTable(w io.Writer, hasHeader bool) *taskFormatterTable {
	t := &taskFormatterTable{
		table: tablewriter.NewWriter(w),
	}
	if hasHeader {
		t.table.SetHeader(taskFormatterColumns)
	}
	t.table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	return t
}

func (t *taskFormatterTable) AddTask(task types.Task) {
	t.table.Append(taskToColumns(task))
}

func (t *taskFormatterTable) Close() {
	t.table.Render()
}

type taskFormatterTSV struct {
	w io.Writer
}

func newTaskFormatterTSV(w io.Writer, header bool) *taskFormatterTSV {
	t := &taskFormatterTSV{w: w}
	if header {
		fmt.Fprintln(t.w, strings.Join(taskFormatterColumns, "\t"))
	}
	return t
}

func (t *taskFormatterTSV) AddTask(task types.Task) {
	fmt.Fprintln(t.w, strings.Join(taskToColumns(task), "\t"))
}

func (t *taskFormatterTSV) Close() {
}

type taskFormatterJSON struct {
	w io.Writer
}

func newTaskFormatterJSON(w io.Writer) *taskFormatterJSON {
	return &taskFormatterJSON{w: w}
}

func (t *taskFormatterJSON) AddTask(task types.Task) {
	// TODO
	b, _ := json.Marshal(task)
	bb := new(bytes.Buffer)
	json.Indent(bb, b, "", "  ")
	t.w.Write(bb.Bytes())
}

func (t *taskFormatterJSON) Close() {
}
