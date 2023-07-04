package ecsta

import "io"

type FormatterOption formatterOption

func NewTaskFormatter(w io.Writer, opt FormatterOption) (taskFormatter, error) {
	return newTaskFormatter(w, formatterOption(opt))
}

var (
	LoadConfig   = loadConfig
	SaveConfig   = saveConfig
	SetConfigDir = setConfigDir
)
