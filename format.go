package ecsta

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/itchyny/gojq"
	"github.com/olekukonko/tablewriter"
)

type formatterOption struct {
	hasHeader bool
	jqQuery   string
}

func newTaskFormatter(w io.Writer, t string, opt formatterOption) (taskFormatter, error) {
	switch t {
	case "table":
		return newTaskFormatterTable(w, opt)
	case "tsv":
		return newTaskFormatterTSV(w, opt)
	case "json":
		return newTaskFormatterJSON(w, opt)
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

func newTaskFormatterTable(w io.Writer, opt formatterOption) (*taskFormatterTable, error) {
	t := &taskFormatterTable{
		table: tablewriter.NewWriter(w),
	}
	if opt.hasHeader {
		t.table.SetHeader(taskFormatterColumns)
	}
	t.table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	return t, nil
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

func newTaskFormatterTSV(w io.Writer, opt formatterOption) (*taskFormatterTSV, error) {
	t := &taskFormatterTSV{w: w}
	if opt.hasHeader {
		fmt.Fprintln(t.w, strings.Join(taskFormatterColumns, "\t"))
	}
	return t, nil
}

func (t *taskFormatterTSV) AddTask(task types.Task) {
	fmt.Fprintln(t.w, strings.Join(taskToColumns(task), "\t"))
}

func (t *taskFormatterTSV) Close() {
}

type taskFormatterJSON struct {
	w    io.Writer
	gojq *gojq.Query
}

func newTaskFormatterJSON(w io.Writer, opt formatterOption) (*taskFormatterJSON, error) {
	f := &taskFormatterJSON{w: w}
	if opt.jqQuery != "" {
		query, err := gojq.Parse(opt.jqQuery)
		if err != nil {
			return nil, err
		}
		f.gojq = query
	}
	return f, nil
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

func UnmarshalJSONForStruct(src []byte, v interface{}) error {
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
		m[fn(key)] = value
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
