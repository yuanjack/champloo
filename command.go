package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ShellSession struct {
	SessionId      string
	CommandCount   int
	DeployId       int
	ExecuteResult  map[Server]ShellCommand
	IsComplete     bool // 是否全部执行完成
	IsCancel       bool //  是否已取消
	Success        bool
	ExecutedCmdNum int // 已执行的命令数
}

func NewShellSession(servers []Server, cmd ShellCommand, deployId int) *ShellSession {
	new := ShellSession{}
	new.CommandCount = cmd.Count()
	new.SessionId = time.Now().Format("s_20060102150405")
	new.ExecuteResult = map[Server]ShellCommand{}
	new.DeployId = deployId
	for _, server := range servers {
		new.ExecuteResult[server] = cmd.Duplicate()
	}

	return &new
}

func (s *ShellSession) Run() {
	for i := 0; i < s.CommandCount; i++ {
		allServerDisable := true
		for server, shell := range s.ExecuteResult {
			// 已停用服务器不处理
			if server.Disable {
				continue
			}

			allServerDisable = false
			cmd := shell.cmds[i]
			cmd.Run(server.Ip, server.Port)
			if cmd.Halt() {
				s.IsComplete = true
				s.Success = false
				return
			}
		}

		if allServerDisable || s.IsCancel {
			s.IsComplete = true
			s.Success = false
			return
		}

	}
	s.Success = true
	s.IsComplete = true
}

func (s *ShellSession) ParallelRun() {
	for i := 0; i < s.CommandCount; i++ {
		isHalt := false
		allServerDisable := true
		var wg sync.WaitGroup
		for server, shell := range s.ExecuteResult {
			// 已停用服务器不处理
			if server.Disable {
				fmt.Println(server.Ip + "已停用.")
				continue
			}

			wg.Add(1)
			allServerDisable = false
			srv := server
			cmd := &shell.cmds[i]
			go func() {
				cmd.Run(srv.Ip, srv.Port)
				if cmd.Halt() {
					isHalt = true
				}

				defer wg.Done()
			}()
		}

		wg.Wait()

		//  判断是否有命令出错，需中断执行
		if isHalt || allServerDisable || s.IsCancel {
			s.IsComplete = true
			s.Success = false
			return
		}
	}
	s.Success = true
	s.IsComplete = true
}

func (s *ShellSession) Cancel() {
	s.IsCancel = true
}

func (s *ShellSession) Output() string {
	output := ""
	isCompelete := s.IsComplete

	// 命令都执行成功时，只显示最后一台服务器的输出信息
	// 命令出错时，显示所有出错服务器输出信息
	for i := 0; i < s.CommandCount; i++ {
		allServerSuccess := true
		allServerDisable := true
		cmdstr := ""
		outputstr := ""
		errorstr := ""
		for server, shell := range s.ExecuteResult {
			// 已停用服务器不处理
			if server.Disable {
				continue
			}

			allServerDisable = false
			cmd := shell.cmds[i]
			if !cmd.HasExecute() {
				continue
			}

			cmdstr = cmd.cmd
			if cmd.success {
				outputstr = cmd.output
			} else {
				errorstr += fmt.Sprintf("<span class='server'>[%s]</span> <span class='error'>%s</span>\n", server.Ip, cmd.output)
				allServerSuccess = false
			}
		}

		if allServerDisable {
			output = "当前没有可用服务器需要部署."
			break
		}

		if cmdstr != "" {
			output += fmt.Sprintln("<i></i><span>" + cmdstr + "</span>")
			output += fmt.Sprintln(outputstr)
		}
		if errorstr != "" {
			output += fmt.Sprintln(errorstr)
		}
		if !allServerSuccess {
			break
		}
	}

	if output != "" {
		serverstr := "[提示] 将更新到如下服务器：\n      "
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
			disablestr = "\n      停用不更新的服务器：\n      " + disablestr
		}

		output = "<span class='tip'>" + serverstr + disablestr + "</span>\n\n" + output

		if isCompelete {
			if s.Success {
				output += "\n\n<span class='success'>已成功部署更新 :)</span>"
			} else {
				output += "\n\n<span class='error'>部署出错 ！！已中止后面步骤执行.</span>"
			}
		}
	}

	return output
}

// 失败时删除已部署的所有文件
func (s *ShellSession) ClearDeploy(dest string) {
	c := command{
		cmd:     fmt.Sprintf("rm -rf %s", dest),
		canHalt: true,
	}

	for server, _ := range s.ExecuteResult {
		// 已停用服务器忽略
		if server.Disable {
			continue
		}

		c.Run(server.Ip, server.Port)
		if c.err != nil {
			fmt.Println(c.err)
		}
	}
}

// 取git最近5个提交日志
func (s *ShellSession) RetrieveGitCommitLog(currentDir string) (CommitLog, error) {
	cmd := `
            cd ` + currentDir + `
            git log --pretty=format:"%h@@@@%an@@@@%s@@@@%ad" -5
             `
	c := command{
		cmd:     cmd,
		canHalt: true,
	}

	output := ""
	for server, _ := range s.ExecuteResult {
		// 已停用服务器忽略
		if server.Disable {
			continue
		}
		c.Run(server.Ip, server.Port)
		if c.err == nil {
			output = c.output
		} else {
			fmt.Println(c.err)
		}
		break
	}

	var commitLog CommitLog
	var err error
	if output != "" {
		// 返回结果是分隔格式
		commitLog.LogEntries = []CommitLogEntry{}
		output = strings.TrimSpace(output)
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			arr := strings.Split(line, "@@@@")
			commitDate, err := time.Parse("Mon Jan 2 15:04:05 2006 -0700", arr[3])
			if err == nil {
				commitLog.LogEntries = append(commitLog.LogEntries, CommitLogEntry{
					Revision: arr[0],
					Author:   arr[1],
					Msg:      arr[2],
					Date:     commitDate,
				})
			} else {
				fmt.Println(err)
				commitLog.LogEntries = append(commitLog.LogEntries, CommitLogEntry{
					Revision: arr[0],
					Author:   arr[1],
					Msg:      arr[2],
				})
			}
		}
	}

	return commitLog, err
}

// 取svn最近5个提交日志
func (s *ShellSession) RetrieveSvnCommitLog(currentDir string, username string, password string) (CommitLog, error) {
	cmd := `
            cd %s
            svn log -l 5 --xml  --username %s --password %s --no-auth-cache
             `
	c := command{
		cmd:     fmt.Sprintf(cmd, currentDir, username, password),
		canHalt: true,
	}

	output := ""
	for server, _ := range s.ExecuteResult {
		// 已停用服务器忽略
		if server.Disable {
			continue
		}
		c.Run(server.Ip, server.Port)
		if c.err == nil {
			output = c.output
		} else {
			fmt.Println(c.err)
		}
		break
	}

	var commitLog CommitLog
	var err error
	if output != "" {
		xmloutput := strings.TrimSpace(output)
		err = xml.Unmarshal([]byte(xmloutput), &commitLog)
		if err != nil {
			fmt.Println(err)
		}
	}

	return commitLog, err
}

type ShellCommand struct {
	cmds []command
}

func NewShellCommand() *ShellCommand {
	new := ShellCommand{}
	new.cmds = []command{}
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
	s.cmds = append(s.cmds, c)
	return s
}

func (s *ShellCommand) Rm(path string) *ShellCommand {
	c := command{
		cmd:     fmt.Sprintf("rm -rf %s", path),
		canHalt: true,
	}
	s.cmds = append(s.cmds, c)
	return s
}

func (s *ShellCommand) Copy(src string, dest string) *ShellCommand {
	cmd := `
            if [ -d "%s" ]; then
                cp -a %s/.  %s
            else
                echo "复制失败，源目录%s不存在."
                exit 1
            fi
             `
	c := command{
		cmd:     fmt.Sprintf(cmd, src, src, dest, src),
		canHalt: true,
	}
	s.cmds = append(s.cmds, c)
	return s
}

func (s *ShellCommand) CopyNoHalt(src string, dest string) *ShellCommand {
	cmd := `
            if [ -d "%s" ]; then
                cp -a %s/.  %s
            fi
             `
	c := command{
		cmd:     fmt.Sprintf(cmd, src, src, dest),
		canHalt: false,
	}
	s.cmds = append(s.cmds, c)
	return s
}

func (s *ShellCommand) Git(dest string, repo string) *ShellCommand {
	c := command{
		cmd:     fmt.Sprintf("git clone %s %s", repo, dest),
		canHalt: true,
	}
	s.cmds = append(s.cmds, c)
	return s
}

func (s *ShellCommand) GitCopyUpdate(currentDir string, dest string, repo string) *ShellCommand {
	cmd := `
            if [ -d "%s" ]; then
                cd %s
                git remote update
            else
                git clone %s %s
            fi
             `
	c := command{
		cmd:     fmt.Sprintf(cmd, currentDir, dest, repo, dest),
		canHalt: true,
	}
	s.cmds = append(s.cmds, c)
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
	s.cmds = append(s.cmds, c)
	return s
}

func (s *ShellCommand) Svn(dest string, repo string, username string, password string) *ShellCommand {
	c := command{
		cmd:     fmt.Sprintf("svn checkout --username %s --password %s --no-auth-cache %s %s", username, password, repo, dest),
		canHalt: true,
	}
	s.cmds = append(s.cmds, c)
	return s
}

func (s *ShellCommand) SvnCopyUpdate(currentDir string, dest string, repo string, username string, password string) *ShellCommand {
	cmd := `
            if [ -d "%s" ]; then
                cd %s
                svn up --username %s --password %s --no-auth-cache
            else
                svn checkout --username %s --password %s --no-auth-cache %s %s
            fi
             `
	c := command{
		cmd:     fmt.Sprintf(cmd, currentDir, dest, username, password, username, password, repo, dest),
		canHalt: true,
	}
	s.cmds = append(s.cmds, c)
	return s
}

func (s *ShellCommand) SvnUpdate(currentDir string, dest string, repo string, username string, password string) *ShellCommand {
	cmd := `
            if [ -d "%s" ]; then
                cd %s
                svn up --username %s --password %s --no-auth-cache
            else
                svn checkout --username %s --password %s --no-auth-cache %s %s
            fi
             `
	c := command{
		cmd:     fmt.Sprintf(cmd, currentDir, currentDir, username, password, username, password, repo, dest),
		canHalt: true,
	}
	s.cmds = append(s.cmds, c)
	return s
}

// cp -Rpn 是为了同步共享目录中新增的文件，但不覆盖已有文件
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

                    cp -Rpf  --preserve=all %s %s
                else
                    yes n|cp -RLi --preserve=all %s %s &>/dev/null
                fi

                rm -rf %s
                ln -s %s %s
                `
	c := command{
		cmd:     fmt.Sprintf(cmd, dest, src, src, src, shared, src, shared, src, dest, src),
		canHalt: true,
	}
	s.cmds = append(s.cmds, c)
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
	s.cmds = append(s.cmds, c)
	return s
}

func (s *ShellCommand) ExistDir(dir string) *ShellCommand {
	cmd := `
            if [ ! -d "%s" ]; then
                    echo "目录%s不存在."
                    exit 1
            fi
            `
	c := command{
		cmd:     fmt.Sprintf(cmd, dir, dir),
		canHalt: true,
	}
	s.cmds = append(s.cmds, c)
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
	s.cmds = append(s.cmds, c)
	return s
}

func (s *ShellCommand) Exec(cmd string, workdir string) *ShellCommand {
	c := command{
		cmd:     cmd,
		workdir: workdir,
		canHalt: true,
	}
	s.cmds = append(s.cmds, c)
	return s
}

func (s *ShellCommand) Ln(src string, dest string) *ShellCommand {
	c := command{
		cmd:     fmt.Sprintf("ln -sfn  %s %s", src, dest),
		canHalt: true,
	}
	s.cmds = append(s.cmds, c)
	return s
}

func (s *ShellCommand) Intro(intro string) *ShellCommand {
	if len(s.cmds) > 0 {
		s.cmds[len(s.cmds)-1].intro = intro
	}
	return s
}

func (s *ShellCommand) Duplicate() ShellCommand {
	sh := NewShellCommand()
	for _, c := range s.cmds {
		sh.cmds = append(sh.cmds, command{
			cmd:        c.cmd,
			intro:      c.intro,
			output:     c.output,
			hasExecute: c.hasExecute,
			success:    c.success,
			canHalt:    c.canHalt,
			err:        c.err,
			workdir:    c.workdir,
		})
	}

	return *sh
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
	if c.canHalt {
		return !c.success || c.err != nil
	}

	return false
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
		c.success = false
		return
	}
	if statusCode != 200 {
		c.hasExecute = true
		c.err = fmt.Errorf("执行脚本请求出错.%s 状态码：%d 内容：%s", url, statusCode, body)
		c.output = c.err.Error()
		c.success = false
		return
	}
	var result ActionMessage
	err = json.Unmarshal([]byte(body), &result)
	if err != nil {
		c.hasExecute = true
		c.err = err
		c.success = false
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

type CommitLog struct {
	LogEntries []CommitLogEntry `xml:"logentry"`
}

type CommitLogEntry struct {
	Revision string    `xml:"revision,attr"`
	Author   string    `xml:"author"`
	Date     time.Time `xml:"date"`
	Msg      string    `xml:"msg"`
}
