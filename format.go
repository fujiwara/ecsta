package ecsta

import (
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
	b, err := MarshalJSONForAPI(task)
	if err != nil {
		panic(err)
	}
	t.w.Write(b)
	t.w.Write([]byte{'\n'})
}

func (t *taskFormatterJSON) Close() {
}

func MarshalJSONForAPI(v interface{}) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	walkMap(m, jsonKeyForAPI)
	return json.MarshalIndent(m, "", "  ")
}

func UnmarshalJSONForAPI(src []byte, v interface{}) error {
	m := map[string]interface{}{}
	if err := json.Unmarshal(src, &m); err != nil {
		return err
	}
	walkMap(m, jsonKeyForStruct)
	if b, err := json.Marshal(m); err != nil {
		return err
	} else {
		return json.Unmarshal(b, v)
	}
}

func jsonKeyForAPI(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func jsonKeyForStruct(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func walkMap(m map[string]interface{}, fn func(string) string) {
	for key, value := range m {
		delete(m, key)
		key = fn(key[:1]) + key[1:]
		m[key] = value
		switch value := value.(type) {
		case map[string]interface{}:
			walkMap(value, fn)
		case []interface{}:
			walkArray(value, fn)
		default:
		}
	}
}

func walkArray(a []interface{}, fn func(string) string) {
	for _, value := range a {
		switch value := value.(type) {
		case map[string]interface{}:
			walkMap(value, fn)
		case []interface{}:
			walkArray(value, fn)
		default:
		}
	}
}
