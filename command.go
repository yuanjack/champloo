package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ShellSession struct {
	SessionId      string
	CommandCount   int
	ExecuteResult  map[Server]ShellCommand
	IsComplete     bool // 是否全部执行完成
	Success        bool
	ExecutedCmdNum int // 已执行的命令数
}

func NewShellSession(servers []Server, cmd ShellCommand) *ShellSession {
	new := ShellSession{}
	new.CommandCount = cmd.Count()
	new.SessionId = time.Now().Format("s_20060102150405")
	new.ExecuteResult = map[Server]ShellCommand{}
	for _, server := range servers {
		new.ExecuteResult[server] = cmd
	}

	return &new
}

func (s *ShellSession) Run() {
	for i := 0; i < s.CommandCount; i++ {
		for server, shell := range s.ExecuteResult {
			// 已停用服务器不处理
			if server.Disable {
				continue
			}

			cmd := shell.cmds[i]
			cmd.Run(server.Ip, server.Port)
			if cmd.Halt() {
				s.IsComplete = true
				s.Success = false
				return
			}
		}
	}
	s.Success = true
	s.IsComplete = true
}

func (s *ShellSession) ParallelRun() {
	for i := 0; i < s.CommandCount; i++ {
		isHalt := false
		var wg sync.WaitGroup
		for server, shell := range s.ExecuteResult {
			// 已停用服务器不处理
			if server.Disable {
				continue
			}

			wg.Add(1)
			go func() {
				cmd := shell.cmds[i]
				cmd.Run(server.Ip, server.Port)
				if cmd.Halt() {
					isHalt = true
				}

				defer wg.Done()
			}()
		}

		wg.Wait()

		//  判断是否有命令出错，需中断执行
		if isHalt {
			s.IsComplete = true
			s.Success = false
			return
		}
	}
	s.Success = true
	s.IsComplete = true
}

func (s *ShellSession) Output() string {
	output := ""

	// 命令都执行成功时，只显示最后一台服务器的输出信息
	// 命令出错时，显示所有出错服务器输出信息
	for i := 0; i < s.CommandCount; i++ {
		allServerSuccess := true
		cmdstr := ""
		outputstr := ""
		for server, shell := range s.ExecuteResult {
			cmd := shell.cmds[i]
			if !cmd.HasExecute() {
				continue
			}

			cmdstr = cmd.cmd
			if cmd.success {
				outputstr = cmd.output
			} else {
				outputstr += fmt.Sprintf("<span class='server'>[%s]</span> <span class='error'>%s</span>\n", server.Ip, cmd.output)
				allServerSuccess = false
			}
		}

		output += fmt.Sprintln("<i></i><span>" + cmdstr + "</span>")
		output += fmt.Sprintln(outputstr)
		if !allServerSuccess {
			break
		}
	}

	if output != "" {
		serverstr := fmt.Sprintln("[提示] 将更新到如下服务器：")
		for server, _ := range s.ExecuteResult {
			if server.Disable {
				continue
			}
			serverstr += server.Ip + ","
		}

		disablestr := ""
		for server, _ := range s.ExecuteResult {
			if server.Disable {
				disablestr += server.Ip + ","
			}
		}
		if disablestr != "" {
			disablestr = fmt.Sprintln("           停用不更新的服务器：") + disablestr
		}

		output = "<span class='tip'>" + serverstr + disablestr + "</span>\n\n" + output

		if s.IsComplete {
			if s.Success {
				output += "\n\n<span class='success'>已成功部署更新 :)</span>"
			} else {
				output += "\n\n<span class='error'>部署出错 ！！已中止后面步骤执行.</span>"
			}
		}
	}

	return output
}

type ShellCommand struct {
	cmds []*command
}

func NewShellCommand() *ShellCommand {
	new := ShellCommand{}
	new.cmds = []*command{}
	return &new
}

func (s *ShellCommand) Count() int {
	return len(s.cmds)
}

func (s *ShellCommand) Mkdir(dir string) *ShellCommand {
	c := command{
		cmd:     fmt.Sprintf("mkdir -p %s", dir),
		canHalt: true,
	}
	s.cmds = append(s.cmds, &c)
	return s
}

func (s *ShellCommand) Rm(path string) *ShellCommand {
	c := command{
		cmd:     fmt.Sprintf("rm -rf %s", path),
		canHalt: true,
	}
	s.cmds = append(s.cmds, &c)
	return s
}

func (s *ShellCommand) Git(dest string, repo string) *ShellCommand {
	c := command{
		cmd:     fmt.Sprintf("git clone %s %s", repo, dest),
		canHalt: true,
	}
	s.cmds = append(s.cmds, &c)
	return s
}

func (s *ShellCommand) GitCopy(currentDir string, dest string, repo string) *ShellCommand {
	cmd := `
            if [ -d "%s" ]; then
                cp -rL %s/*  %s
                cd %s
                git remote update
            else
                git clone %s %s
            fi
             `
	c := command{
		cmd:     fmt.Sprintf(cmd, currentDir, currentDir, dest, dest, repo, dest),
		canHalt: true,
	}
	s.cmds = append(s.cmds, &c)
	return s
}

func (s *ShellCommand) GitUpdate(currentDir string, dest string, repo string) *ShellCommand {
	cmd := `
            if [ -d "%s" ]; then
                cd %s
                git remote update
            else
                git clone %s %s
            fi
             `
	c := command{
		cmd:     fmt.Sprintf(cmd, currentDir, currentDir, repo, dest),
		canHalt: true,
	}
	s.cmds = append(s.cmds, &c)
	return s
}

func (s *ShellCommand) Svn(dest string, repo string, username string, password string) *ShellCommand {
	c := command{
		cmd:     fmt.Sprintf("svn checkout --username %s --password %s  %s %s", username, password, repo, dest),
		canHalt: true,
	}
	s.cmds = append(s.cmds, &c)
	return s
}

func (s *ShellCommand) SvnCopy(currentDir string, dest string, repo string, username string, password string) *ShellCommand {
	cmd := `
            if [ -d "%s" ]; then
                cp -rL %s/*  %s
                cd %s
                svn up
            else
                svn checkout --username %s --password %s  %s %s
            fi
             `
	c := command{
		cmd:     fmt.Sprintf(cmd, currentDir, currentDir, dest, dest, username, password, repo, dest),
		canHalt: true,
	}
	s.cmds = append(s.cmds, &c)
	return s
}

func (s *ShellCommand) SvnUpdate(currentDir string, dest string, repo string, username string, password string) *ShellCommand {
	cmd := `
            if [ -d "%s" ]; then
                cd %s
                svn up
            else
                svn checkout --username %s --password %s  %s %s
            fi
             `
	c := command{
		cmd:     fmt.Sprintf(cmd, currentDir, currentDir, username, password, repo, dest),
		canHalt: true,
	}
	s.cmds = append(s.cmds, &c)
	return s
}

func (s *ShellCommand) Shared(srcPath string, sharedDir string) *ShellCommand {
	src := strings.TrimSpace(srcPath)
	shared := strings.TrimSpace(sharedDir)
	name := filepath.Base(src)
	dest := fmt.Sprintf("%s/%s", shared, name)
	cmd := `
                if [ ! -d "%s" ]; then
                    if [ ! -d "%s" ]; then
                            echo "共享的目录%s不存在."
                            exit 1
                    fi

                    cp -R  -p  -f %s %s
                fi

                rm -rf %s
                ln -s %s %s
                `
	c := command{
		cmd:     fmt.Sprintf(cmd, dest, src, src, src, shared, src, dest, src),
		canHalt: true,
	}
	s.cmds = append(s.cmds, &c)
	return s
}

func (s *ShellCommand) ClearBackup(dir string, leaveNum int) *ShellCommand {
	cmd := `
            i=0
            for p in $(ls -dr %s/*)
            do

            if [ -d "$p" ]; then
                i=$(($i+1))

                if [ $i -gt  %d ]; then
                     rm -rf $p
                     echo "$p has been removed"
                fi
            fi

            done
            `
	c := command{
		cmd:     fmt.Sprintf(cmd, dir, leaveNum),
		canHalt: true,
	}
	s.cmds = append(s.cmds, &c)
	return s
}

func (s *ShellCommand) Rollback(src string, dest string) *ShellCommand {
	cmd := `
            if [ ! -d "%s" ]; then
                    echo "版本目录%s不存在."
                    exit 1
            fi

             ln -sfn  %s %s
            `
	c := command{
		cmd:     fmt.Sprintf(cmd, src, src, src, dest),
		canHalt: true,
	}
	s.cmds = append(s.cmds, &c)
	return s
}

func (s *ShellCommand) Exec(cmd string, workdir string) *ShellCommand {
	c := command{
		cmd:     cmd,
		workdir: workdir,
		canHalt: true,
	}
	s.cmds = append(s.cmds, &c)
	return s
}

func (s *ShellCommand) Ln(src string, dest string) *ShellCommand {
	c := command{
		cmd:     fmt.Sprintf("ln -sfn  %s %s", src, dest),
		canHalt: true,
	}
	s.cmds = append(s.cmds, &c)
	return s
}

func (s *ShellCommand) Intro(intro string) *ShellCommand {
	if len(s.cmds) > 0 {
		s.cmds[len(s.cmds)-1].intro = intro
	}
	return s
}

// 对shell命令的封装
type command struct {
	cmd        string // 需执行的命令
	intro      string // 命令介绍
	output     string //  执行命令结果
	hasExecute bool   // 是否已执行
	success    bool   // 是否执行成功
	canHalt    bool   // 命令执行失败是否挂起后面命令执行
	err        error  // 执行错误
	workdir    string // 执行目录
}

func (c *command) Introduction() string {
	return c.intro
}

func (c *command) HasExecute() bool {
	return c.hasExecute
}

func (c *command) Output() string {
	return c.output
}

func (c *command) Success() bool {
	return c.success
}
func (c *command) Halt() bool {
	return !c.success && c.canHalt
}

func (c *command) Error() error {
	return c.err
}

func (c *command) Run(ip string, port int) {
	cmd := strings.Replace(c.cmd, "\r\n", "\n", -1)

	// 请求接口执行命令
	jsonCmd := JsonCommand{
		Dir: c.workdir,
		Cmd: cmd,
	}

	url := fmt.Sprintf("http://%s:%d/run", ip, port)
	body, statusCode, err := PostJson(url, jsonCmd)

	if *debug {
		fmt.Println("执行脚本：" + url)
		fmt.Println(cmd)
		fmt.Printf("执行结果：%d %v %s \n", statusCode, err, body)
		fmt.Println()
	}
	if err != nil {
		c.hasExecute = true
		c.err = err
		c.output = err.Error()
		return
	}
	if statusCode != 200 {
		c.hasExecute = true
		c.err = fmt.Errorf("执行脚本请求出错.%s 状态码：%d 内容：%s", url, statusCode, body)
		c.output = c.err.Error()
		return
	}
	var result ActionMessage
	err = json.Unmarshal([]byte(body), &result)
	if err != nil {
		c.hasExecute = true
		c.err = err
		return
	}
	c.hasExecute = true
	c.success = result.Success
	c.output = result.Data.(string)
	if c.output == "" {
		c.output = result.Message
	}
}

type JsonCommand struct {
	Dir string `json:"dir"`
	Cmd string `json:"cmd"`
}
