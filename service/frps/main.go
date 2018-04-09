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

package main

import (
	"os"
	"path/filepath"

	ini "github.com/vaughan0/go-ini"

	"github.com/fatedier/frp/models/config"
	"github.com/fatedier/frp/server"
	"github.com/fatedier/frp/utils/log"
	"github.com/kardianos/service"
)

var logger service.Logger

type program struct{}

func (p *program) Start(s service.Service) error {
	// Start should not block. Do the actual work async.
	go p.run()
	return nil
}

func (p *program) run() {
	// Do work here
	var err error
	fileDir := filepath.Dir(os.Args[0])
	confFile := filepath.Join(fileDir, "frps.ini")
	logger.Infof("get config file path:%s", confFile)

	conf, err := ini.LoadFile(confFile)
	if err != nil {
		logger.Info(err)
		os.Exit(1)
	}
	config.ServerCommonCfg, err = config.LoadServerCommonConf(conf)
	if err != nil {
		logger.Info(err)
		os.Exit(1)
	}

	log.InitLog(config.ServerCommonCfg.LogWay, config.ServerCommonCfg.LogFile,
		config.ServerCommonCfg.LogLevel, config.ServerCommonCfg.LogMaxDays)

	svr, err := server.NewService()
	if err != nil {
		logger.Info(err)
		os.Exit(1)
	}
	log.Info("Start frps success")
	if config.ServerCommonCfg.PrivilegeMode == true {
		log.Info("PrivilegeMode is enabled, you should pay more attention to security issues")
	}
	server.ServerService = svr
	svr.Run()
}

func (p *program) Stop(s service.Service) error {
	// Stop should not block. Return with a few seconds.
	return nil
}

func main() {
	svcConfig := &service.Config{
		Name:        "frps service",
		DisplayName: "frps service",
		Description: "config file frps.ini",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Error(err.Error())
	}
	logger, err = s.Logger(nil)
	if err != nil {
		log.Error(err.Error())
	}

	if len(os.Args) == 2 { //如果有命令则执行
		err = service.Control(s, os.Args[1])
		if err != nil {
			logger.Error(err)
		}
	} else { //否则说明是方法启动了
		err = s.Run()
		if err != nil {
			logger.Error(err)
		}
	}
	if err != nil {
		logger.Error(err)
	}
}
