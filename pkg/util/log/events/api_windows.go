// Copyright 2016 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package events

import (
	"golang.org/x/sys/windows/registry"
	"os"
)

func SourceExists(serviceName string) bool {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, "SYSTEM\\CurrentControlSet\\Services\\EventLog\\Frp\\" + serviceName, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer func(key registry.Key) {
		_ = key.Close()
	}(key)
	value, _, err := key.GetStringValue("EventMessageFile")
	if err != nil {
		return false
	}
	stat, err := os.Stat(value)
	if err != nil {
		return false
	}
	return !stat.IsDir()
}

func CreateEventSource(serviceName string) error {
	msgFile, err := getEventMessageFile()
	if err != nil {
		return err
	}
	eventKey, err := registry.OpenKey(registry.LOCAL_MACHINE, "SYSTEM\\CurrentControlSet\\Services\\EventLog", registry.CREATE_SUB_KEY)
	if err != nil {
		return err
	}
	defer func(eventKey registry.Key) {
		_ = eventKey.Close()
	}(eventKey)

	frpKey, _, err := registry.CreateKey(eventKey, "Frp", registry.CREATE_SUB_KEY)
	if err != nil {
		return err
	}
	defer func(frpKey registry.Key) {
		_ = frpKey.Close()
	}(frpKey)

	srcKey, _, err := registry.CreateKey(frpKey, serviceName, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer func(srcKey registry.Key) {
		_ = srcKey.Close()
	}(srcKey)

	err = srcKey.SetExpandStringValue("EventMessageFile", msgFile)
	if err != nil {
		return err
	}
	return nil
}

func DeleteEventSource(serviceName string) error {
	err := registry.DeleteKey(registry.LOCAL_MACHINE, "SYSTEM\\CurrentControlSet\\Services\\EventLog\\Frp\\" + serviceName)
	if err != nil {
		return err
	}

	// Delete the whole tree if nothing but "Frp" is left. Simply abort if any error occurred
	frpKey, err := registry.OpenKey(registry.LOCAL_MACHINE, "SYSTEM\\CurrentControlSet\\Services\\EventLog\\Frp", registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil
	}
	defer func(frpKey registry.Key) {
		_ = frpKey.Close()
	}(frpKey)

	keyNames, err := frpKey.ReadSubKeyNames(0)
	if err != nil {
		return nil
	}
	if len(keyNames) == 1 && keyNames[0] == "Frp" {
		err := registry.DeleteKey(registry.LOCAL_MACHINE, "SYSTEM\\CurrentControlSet\\Services\\EventLog\\Frp")
		if err != nil {
			return nil
		}
	}
	return nil
}

func getEventMessageFile() (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, "SOFTWARE\\Microsoft\\NET Framework Setup\\NDP\\v4\\Client", registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	path, _, err := key.GetStringValue("InstallPath")
	if err != nil {
		return "", err
	}
	return path + "EventLogMessages.dll", nil
}