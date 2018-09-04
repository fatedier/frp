package alils

type InputDetail struct {
	LogType       string   `json:"logType"`
	LogPath       string   `json:"logPath"`
	FilePattern   string   `json:"filePattern"`
	LocalStorage  bool     `json:"localStorage"`
	TimeFormat    string   `json:"timeFormat"`
	LogBeginRegex string   `json:"logBeginRegex"`
	Regex         string   `json:"regex"`
	Keys          []string `json:"key"`
	FilterKeys    []string `json:"filterKey"`
	FilterRegex   []string `json:"filterRegex"`
	TopicFormat   string   `json:"topicFormat"`
}

type OutputDetail struct {
	Endpoint     string `json:"endpoint"`
	LogStoreName string `json:"logstoreName"`
}

type LogConfig struct {
	Name         string       `json:"configName"`
	InputType    string       `json:"inputType"`
	InputDetail  InputDetail  `json:"inputDetail"`
	OutputType   string       `json:"outputType"`
	OutputDetail OutputDetail `json:"outputDetail"`

	CreateTime     uint32
	LastModifyTime uint32

	project *LogProject
}

// GetAppliedMachineGroup returns applied machine group of this config.
func (c *LogConfig) GetAppliedMachineGroup(confName string) (groupNames []string, err error) {
	groupNames, err = c.project.GetAppliedMachineGroups(c.Name)
	return
}
