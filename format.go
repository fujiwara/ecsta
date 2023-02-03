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
	Format       string
	HasHeader    bool
	AppendTaskID bool
	Query        string
}

type taskFormatterFunc func(io.Writer, formatterOption) (taskFormatter, error)

var taskFormatters map[string]taskFormatterFunc = map[string]taskFormatterFunc{
	"table": newTaskFormatterTable,
	"tsv":   newTaskFormatterTSV,
	"json":  newTaskFormatterJSON,
}

func newTaskFormatter(w io.Writer, opt formatterOption) (taskFormatter, error) {
	if f, ok := taskFormatters[opt.Format]; ok {
		return f(w, opt)
	}
	return nil, fmt.Errorf("unknown task formatter: %s", opt.Format)
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

func newTaskFormatterTable(w io.Writer, opt formatterOption) (taskFormatter, error) {
	t := &taskFormatterTable{
		table: tablewriter.NewWriter(w),
	}
	if opt.HasHeader {
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

func newTaskFormatterTSV(w io.Writer, opt formatterOption) (taskFormatter, error) {
	t := &taskFormatterTSV{w: w}
	if opt.HasHeader {
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
	w            io.Writer
	gojq         *gojq.Query
	appendTaskID bool
}

func newTaskFormatterJSON(w io.Writer, opt formatterOption) (taskFormatter, error) {
	f := &taskFormatterJSON{
		w:            w,
		appendTaskID: opt.AppendTaskID,
	}
	if opt.Query != "" {
		query, err := gojq.Parse(opt.Query)
		if err != nil {
			return nil, err
		}
		f.gojq = query
	}
	return f, nil
}

func (t *taskFormatterJSON) AddTask(task types.Task) {
	b, err := MarshalJSONForAPI(task, t.gojq)
	if err != nil {
		panic(err)
	}
	if t.appendTaskID {
		// ensure task arn at the beginning of the line
		io.WriteString(t.w, arnToName(*task.TaskArn)+"\t")
	}
	t.w.Write(b)
	t.w.Write([]byte{'\n'})
}

func (t *taskFormatterJSON) Close() {
}

func MarshalJSONForAPI(v interface{}, query *gojq.Query) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	walkMap(m, jsonKeyForAPI)
	if query == nil {
		return json.MarshalIndent(m, "", "  ")
	}
	iter := query.Run(m)
	for {
		v, ok := iter.Next()
		if !ok {
			return nil, nil // no output(or end of stream)
		}
		if err, ok := v.(error); ok {
			return nil, err
		}
		switch val := v.(type) {
		case string:
			return []byte(val), nil
		default:
			return json.Marshal(val) // without indent
		}
	}
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
