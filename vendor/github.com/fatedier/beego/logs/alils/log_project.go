/*
Package sls implements the SDK(v0.5.0) of Simple Log Service(abbr. SLS).

For more description about SLS, please read this article:
http://gitlab.alibaba-inc.com/sls/doc.
*/
package alils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
)

// Error message in SLS HTTP response.
type errorMessage struct {
	Code    string `json:"errorCode"`
	Message string `json:"errorMessage"`
}

type LogProject struct {
	Name            string // Project name
	Endpoint        string // IP or hostname of SLS endpoint
	AccessKeyId     string
	AccessKeySecret string
}

// NewLogProject creates a new SLS project.
func NewLogProject(name, endpoint, accessKeyId, accessKeySecret string) (p *LogProject, err error) {
	p = &LogProject{
		Name:            name,
		Endpoint:        endpoint,
		AccessKeyId:     accessKeyId,
		AccessKeySecret: accessKeySecret,
	}
	return p, nil
}

// ListLogStore returns all logstore names of project p.
func (p *LogProject) ListLogStore() (storeNames []string, err error) {
	h := map[string]string{
		"x-sls-bodyrawsize": "0",
	}

	uri := fmt.Sprintf("/logstores")
	r, err := request(p, "GET", uri, h, nil)
	if err != nil {
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(buf, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to list logstore")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	type Body struct {
		Count     int
		LogStores []string
	}
	body := &Body{}

	err = json.Unmarshal(buf, body)
	if err != nil {
		return
	}

	storeNames = body.LogStores

	return
}

// GetLogStore returns logstore according by logstore name.
func (p *LogProject) GetLogStore(name string) (s *LogStore, err error) {
	h := map[string]string{
		"x-sls-bodyrawsize": "0",
	}

	r, err := request(p, "GET", "/logstores/"+name, h, nil)
	if err != nil {
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(buf, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to get logstore")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	s = &LogStore{}
	err = json.Unmarshal(buf, s)
	if err != nil {
		return
	}
	s.project = p
	return
}

// CreateLogStore creates a new logstore in SLS,
// where name is logstore name,
// and ttl is time-to-live(in day) of logs,
// and shardCnt is the number of shards.
func (p *LogProject) CreateLogStore(name string, ttl, shardCnt int) (err error) {

	type Body struct {
		Name       string `json:"logstoreName"`
		TTL        int    `json:"ttl"`
		ShardCount int    `json:"shardCount"`
	}

	store := &Body{
		Name:       name,
		TTL:        ttl,
		ShardCount: shardCnt,
	}

	body, err := json.Marshal(store)
	if err != nil {
		return
	}

	h := map[string]string{
		"x-sls-bodyrawsize": fmt.Sprintf("%v", len(body)),
		"Content-Type":      "application/json",
		"Accept-Encoding":   "deflate", // TODO: support lz4
	}

	r, err := request(p, "POST", "/logstores", h, body)
	if err != nil {
		return
	}

	body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(body, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to create logstore")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	return
}

// DeleteLogStore deletes a logstore according by logstore name.
func (p *LogProject) DeleteLogStore(name string) (err error) {
	h := map[string]string{
		"x-sls-bodyrawsize": "0",
	}

	r, err := request(p, "DELETE", "/logstores/"+name, h, nil)
	if err != nil {
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(body, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to delete logstore")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}
	return
}

// UpdateLogStore updates a logstore according by logstore name,
// obviously we can't modify the logstore name itself.
func (p *LogProject) UpdateLogStore(name string, ttl, shardCnt int) (err error) {

	type Body struct {
		Name       string `json:"logstoreName"`
		TTL        int    `json:"ttl"`
		ShardCount int    `json:"shardCount"`
	}

	store := &Body{
		Name:       name,
		TTL:        ttl,
		ShardCount: shardCnt,
	}

	body, err := json.Marshal(store)
	if err != nil {
		return
	}

	h := map[string]string{
		"x-sls-bodyrawsize": fmt.Sprintf("%v", len(body)),
		"Content-Type":      "application/json",
		"Accept-Encoding":   "deflate", // TODO: support lz4
	}

	r, err := request(p, "PUT", "/logstores", h, body)
	if err != nil {
		return
	}

	body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(body, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to update logstore")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	return
}

// ListMachineGroup returns machine group name list and the total number of machine groups.
// The offset starts from 0 and the size is the max number of machine groups could be returned.
func (p *LogProject) ListMachineGroup(offset, size int) (m []string, total int, err error) {
	h := map[string]string{
		"x-sls-bodyrawsize": "0",
	}

	if size <= 0 {
		size = 500
	}

	uri := fmt.Sprintf("/machinegroups?offset=%v&size=%v", offset, size)
	r, err := request(p, "GET", uri, h, nil)
	if err != nil {
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(buf, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to list machine group")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	type Body struct {
		MachineGroups []string
		Count         int
		Total         int
	}
	body := &Body{}

	err = json.Unmarshal(buf, body)
	if err != nil {
		return
	}

	m = body.MachineGroups
	total = body.Total

	return
}

// GetMachineGroup retruns machine group according by machine group name.
func (p *LogProject) GetMachineGroup(name string) (m *MachineGroup, err error) {
	h := map[string]string{
		"x-sls-bodyrawsize": "0",
	}

	r, err := request(p, "GET", "/machinegroups/"+name, h, nil)
	if err != nil {
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(buf, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to get machine group:%v", name)
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	m = &MachineGroup{}
	err = json.Unmarshal(buf, m)
	if err != nil {
		return
	}
	m.project = p
	return
}

// CreateMachineGroup creates a new machine group in SLS.
func (p *LogProject) CreateMachineGroup(m *MachineGroup) (err error) {

	body, err := json.Marshal(m)
	if err != nil {
		return
	}

	h := map[string]string{
		"x-sls-bodyrawsize": fmt.Sprintf("%v", len(body)),
		"Content-Type":      "application/json",
		"Accept-Encoding":   "deflate", // TODO: support lz4
	}

	r, err := request(p, "POST", "/machinegroups", h, body)
	if err != nil {
		return
	}

	body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(body, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to create machine group")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	return
}

// UpdateMachineGroup updates a machine group.
func (p *LogProject) UpdateMachineGroup(m *MachineGroup) (err error) {

	body, err := json.Marshal(m)
	if err != nil {
		return
	}

	h := map[string]string{
		"x-sls-bodyrawsize": fmt.Sprintf("%v", len(body)),
		"Content-Type":      "application/json",
		"Accept-Encoding":   "deflate", // TODO: support lz4
	}

	r, err := request(p, "PUT", "/machinegroups/"+m.Name, h, body)
	if err != nil {
		return
	}

	body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(body, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to update machine group")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	return
}

// DeleteMachineGroup deletes machine group according machine group name.
func (p *LogProject) DeleteMachineGroup(name string) (err error) {
	h := map[string]string{
		"x-sls-bodyrawsize": "0",
	}

	r, err := request(p, "DELETE", "/machinegroups/"+name, h, nil)
	if err != nil {
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(body, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to delete machine group")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}
	return
}

// ListConfig returns config names list and the total number of configs.
// The offset starts from 0 and the size is the max number of configs could be returned.
func (p *LogProject) ListConfig(offset, size int) (cfgNames []string, total int, err error) {
	h := map[string]string{
		"x-sls-bodyrawsize": "0",
	}

	if size <= 0 {
		size = 100
	}

	uri := fmt.Sprintf("/configs?offset=%v&size=%v", offset, size)
	r, err := request(p, "GET", uri, h, nil)
	if err != nil {
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(buf, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to delete machine group")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	type Body struct {
		Total   int
		Configs []string
	}
	body := &Body{}

	err = json.Unmarshal(buf, body)
	if err != nil {
		return
	}

	cfgNames = body.Configs
	total = body.Total
	return
}

// GetConfig returns config according by config name.
func (p *LogProject) GetConfig(name string) (c *LogConfig, err error) {
	h := map[string]string{
		"x-sls-bodyrawsize": "0",
	}

	r, err := request(p, "GET", "/configs/"+name, h, nil)
	if err != nil {
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(buf, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to delete config")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	c = &LogConfig{}
	err = json.Unmarshal(buf, c)
	if err != nil {
		return
	}
	c.project = p
	return
}

// UpdateConfig updates a config.
func (p *LogProject) UpdateConfig(c *LogConfig) (err error) {

	body, err := json.Marshal(c)
	if err != nil {
		return
	}

	h := map[string]string{
		"x-sls-bodyrawsize": fmt.Sprintf("%v", len(body)),
		"Content-Type":      "application/json",
		"Accept-Encoding":   "deflate", // TODO: support lz4
	}

	r, err := request(p, "PUT", "/configs/"+c.Name, h, body)
	if err != nil {
		return
	}

	body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(body, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to update config")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	return
}

// CreateConfig creates a new config in SLS.
func (p *LogProject) CreateConfig(c *LogConfig) (err error) {

	body, err := json.Marshal(c)
	if err != nil {
		return
	}

	h := map[string]string{
		"x-sls-bodyrawsize": fmt.Sprintf("%v", len(body)),
		"Content-Type":      "application/json",
		"Accept-Encoding":   "deflate", // TODO: support lz4
	}

	r, err := request(p, "POST", "/configs", h, body)
	if err != nil {
		return
	}

	body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(body, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to update config")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	return
}

// DeleteConfig deletes a config according by config name.
func (p *LogProject) DeleteConfig(name string) (err error) {
	h := map[string]string{
		"x-sls-bodyrawsize": "0",
	}

	r, err := request(p, "DELETE", "/configs/"+name, h, nil)
	if err != nil {
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(body, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to delete config")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}
	return
}

// GetAppliedMachineGroups returns applied machine group names list according config name.
func (p *LogProject) GetAppliedMachineGroups(confName string) (groupNames []string, err error) {
	h := map[string]string{
		"x-sls-bodyrawsize": "0",
	}

	uri := fmt.Sprintf("/configs/%v/machinegroups", confName)
	r, err := request(p, "GET", uri, h, nil)
	if err != nil {
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(buf, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to get applied machine groups")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	type Body struct {
		Count         int
		Machinegroups []string
	}

	body := &Body{}
	err = json.Unmarshal(buf, body)
	if err != nil {
		return
	}

	groupNames = body.Machinegroups
	return
}

// GetAppliedConfigs returns applied config names list according machine group name groupName.
func (p *LogProject) GetAppliedConfigs(groupName string) (confNames []string, err error) {
	h := map[string]string{
		"x-sls-bodyrawsize": "0",
	}

	uri := fmt.Sprintf("/machinegroups/%v/configs", groupName)
	r, err := request(p, "GET", uri, h, nil)
	if err != nil {
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(buf, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to applied configs")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	type Cfg struct {
		Count   int      `json:"count"`
		Configs []string `json:"configs"`
	}

	body := &Cfg{}
	err = json.Unmarshal(buf, body)
	if err != nil {
		return
	}

	confNames = body.Configs
	return
}

// ApplyConfigToMachineGroup applies config to machine group.
func (p *LogProject) ApplyConfigToMachineGroup(confName, groupName string) (err error) {
	h := map[string]string{
		"x-sls-bodyrawsize": "0",
	}

	uri := fmt.Sprintf("/machinegroups/%v/configs/%v", groupName, confName)
	r, err := request(p, "PUT", uri, h, nil)
	if err != nil {
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(buf, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to apply config to machine group")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}
	return
}

// RemoveConfigFromMachineGroup removes config from machine group.
func (p *LogProject) RemoveConfigFromMachineGroup(confName, groupName string) (err error) {
	h := map[string]string{
		"x-sls-bodyrawsize": "0",
	}

	uri := fmt.Sprintf("/machinegroups/%v/configs/%v", groupName, confName)
	r, err := request(p, "DELETE", uri, h, nil)
	if err != nil {
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(buf, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to remove config from machine group")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Printf("%s\n", dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}
	return
}
