package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
)

// 默认发布
func ExecuteDeployDefault(username string, params martini.Params, r render.Render) {
	executeDeploy("", username, params, r)
}

// 发布到开发场景(Dev)
func ExecuteDeployDev(username string, params martini.Params, r render.Render) {
	executeDeploy("dev", username, params, r)
}

// 发布到产品场景(Prod)
func ExecuteDeployProd(username string, params martini.Params, r render.Render) {
	executeDeploy("prod", username, params, r)
}

func executeDeploy(stage string, username string, params martini.Params, r render.Render) {
	id, _ := strconv.Atoi(params["id"])

	var conf SystemConfig
	db.First(&conf, id)
	if conf.Way == "update" {
		executeDeployUpdate(stage, username, params, r)
		return
	}

	if isDeploying(id) {
		sendFailMsg(r, "另一部署进程正在进行中，请稍候.", nil)
		return
	}

	workdir := conf.Path
	switch stage {
	case "dev":
		workdir = strings.TrimSuffix(conf.Path, "/") + "/development"
	case "prod":
		workdir = strings.TrimSuffix(conf.Path, "/") + "/production"
	default:
	}
	releaseDir := fmt.Sprintf("%s/releases", workdir)
	sharedDir := fmt.Sprintf("%s/shared", workdir)
	currentDir := fmt.Sprintf("%s/current", workdir)
	tags := conf.Tags
	version := time.Now().Format("20060102150405")
	deploy := Deploy{
		SystemId: id,
		Stage:    stage,
		Operator: string(username),
		Version:  version,
	}
	err := db.Save(&deploy).Error
	if err != nil {
		sendFailMsg(r, "创建部署记录出错."+err.Error(), nil)
		return
	}

	cmds := NewShellCommand()
	// 1.创建部署目录
	cmds.Mkdir(workdir).Mkdir(releaseDir).Mkdir(sharedDir)

	// 2.创建新版本目录
	versionDir := fmt.Sprintf("%s/%s", releaseDir, version)
	cmds.Mkdir(versionDir)

	// 3.checkout代码
	if strings.Contains(conf.Repo, ".git") {
		switch conf.Way {
		case "copy":
			cmds.CopyNoHalt(currentDir, versionDir)

			// 删除旧共享目录，要不很容易因为共享目录代码冲突导致update不了
			if strings.TrimSpace(conf.Shared) != "" {
				paths := strings.Split(conf.Shared, "\n")
				for _, path := range paths {
					sharePath := strings.TrimSpace(path)
					sharePath = strings.Replace(sharePath, "$path", versionDir, -1)

					cmds.Rm(sharePath)
				}
			}

			cmds.GitCopyUpdate(currentDir, versionDir, conf.Repo)
		default:
			cmds.Git(versionDir, conf.Repo)
		}

	} else {
		switch conf.Way {
		case "copy":
			cmds.CopyNoHalt(currentDir, versionDir)

			// 删除旧共享目录，要不很容易因为共享目录代码冲突导致update不了
			if strings.TrimSpace(conf.Shared) != "" {
				paths := strings.Split(conf.Shared, "\n")
				for _, path := range paths {
					sharePath := strings.TrimSpace(path)
					sharePath = strings.Replace(sharePath, "$path", versionDir, -1)

					cmds.Rm(sharePath)
				}
			}
			cmds.SvnCopyUpdate(currentDir, versionDir, conf.Repo, conf.UserName, conf.Password)
		default:
			cmds.Svn(versionDir, conf.Repo, conf.UserName, conf.Password)
		}

	}

	// 4.处理共享目录
	if strings.TrimSpace(conf.Shared) != "" {
		paths := strings.Split(conf.Shared, "\n")
		for _, path := range paths {
			sharePath := strings.TrimSpace(path)
			src := strings.Replace(sharePath, "$path", versionDir, -1)

			cmds.Shared(src, sharedDir)
		}
	}

	// 5.清理备份版本
	cmds.ClearBackup(releaseDir, conf.BackupNum)

	// 6.TODO:测试?

	// 7.执行部署上线前命令
	if strings.TrimSpace(conf.BeforeCmd) != "" {
		conf.BeforeCmd = strings.Replace(conf.BeforeCmd, "$path", versionDir, -1)
		conf.BeforeCmd = strings.Replace(conf.BeforeCmd, "$share", sharedDir, -1)
		conf.BeforeCmd = strings.Replace(conf.BeforeCmd, "$release", releaseDir, -1)
		cmds.Exec(conf.BeforeCmd, versionDir)
	}

	// 执行对应场景上线前命令
	if stage == "dev" && strings.TrimSpace(conf.DevBeforeCmd) != "" {
		conf.DevBeforeCmd = strings.Replace(conf.DevBeforeCmd, "$path", versionDir, -1)
		conf.DevBeforeCmd = strings.Replace(conf.DevBeforeCmd, "$share", sharedDir, -1)
		conf.DevBeforeCmd = strings.Replace(conf.DevBeforeCmd, "$release", releaseDir, -1)
		cmds.Exec(conf.DevBeforeCmd, versionDir)
	}
	if stage == "prod" && strings.TrimSpace(conf.ProdBeforeCmd) != "" {
		conf.ProdBeforeCmd = strings.Replace(conf.ProdBeforeCmd, "$path", versionDir, -1)
		conf.ProdBeforeCmd = strings.Replace(conf.ProdBeforeCmd, "$share", sharedDir, -1)
		conf.ProdBeforeCmd = strings.Replace(conf.ProdBeforeCmd, "$release", releaseDir, -1)
		cmds.Exec(conf.ProdBeforeCmd, versionDir)
	}

	// 8.把current软链接指向新版本
	cmds.Ln(versionDir, currentDir)

	// 11.执行部署后命令
	if strings.TrimSpace(conf.AfterCmd) != "" {
		conf.AfterCmd = strings.Replace(conf.AfterCmd, "$path", versionDir, -1)
		conf.AfterCmd = strings.Replace(conf.AfterCmd, "$share", sharedDir, -1)
		conf.AfterCmd = strings.Replace(conf.AfterCmd, "$release", releaseDir, -1)
		cmds.Exec(conf.AfterCmd, versionDir)
	}

	// 执行对应场景上线后命令
	if stage == "dev" && strings.TrimSpace(conf.DevAfterCmd) != "" {
		conf.DevAfterCmd = strings.Replace(conf.DevAfterCmd, "$path", versionDir, -1)
		conf.DevAfterCmd = strings.Replace(conf.DevAfterCmd, "$share", sharedDir, -1)
		conf.DevAfterCmd = strings.Replace(conf.DevAfterCmd, "$release", releaseDir, -1)
		cmds.Exec(conf.DevAfterCmd, versionDir)
	}
	if stage == "prod" && strings.TrimSpace(conf.ProdAfterCmd) != "" {
		conf.ProdAfterCmd = strings.Replace(conf.ProdAfterCmd, "$path", versionDir, -1)
		conf.ProdAfterCmd = strings.Replace(conf.ProdAfterCmd, "$share", sharedDir, -1)
		conf.ProdAfterCmd = strings.Replace(conf.ProdAfterCmd, "$release", releaseDir, -1)
		cmds.Exec(conf.ProdAfterCmd, versionDir)
	}

	// 取对应该tag的所有服务器
	servers := getTagServers(tags)

	session := NewShellSession(servers, *cmds, deploy.Id)
	mutex.Lock()
	sessions[id] = session
	mutex.Unlock()

	startTime := time.Now()
	session.ParallelRun()
	if !session.Success {
		// 删除部署的文件夹
		session.ClearDeploy(versionDir)

		deploy.Stage = stage
		deploy.Operator = string(username)
		deploy.Status = -1
		deploy.Output = session.Output()
		deploy.ElapsedTime = int(time.Now().Sub(startTime).Seconds())
		db.Save(&deploy)

		sendFailMsg(r, "部署失败.", session.Output())
		return
	}

	// 取commit日志
	revision, commitLog := getCommitLog(session, id, stage, conf, currentDir)
	deploy.Revision = revision
	deploy.CommitLog = commitLog

	// 去掉旧的部署的启用状态
	db.Model(Deploy{}).Where(Deploy{SystemId: id, Stage: stage, Enable: true}).Update(map[string]interface{}{"enable": false})

	deploy.Stage = stage
	deploy.Operator = string(username)
	deploy.Status = 1
	deploy.Output = session.Output()
	deploy.Enable = true
	deploy.ElapsedTime = int(time.Now().Sub(startTime).Seconds())
	db.Save(&deploy)
	sendSuccessMsg(r, nil)
	return

}

func executeDeployUpdate(stage string, username string, params martini.Params, r render.Render) {
	id, _ := strconv.Atoi(params["id"])

	if isDeploying(id) {
		sendFailMsg(r, "另一部署进程正在进行中，请稍候.", nil)
		return
	}

	var conf SystemConfig
	db.First(&conf, id)
	workdir := conf.Path
	switch stage {
	case "dev":
		workdir = strings.TrimSuffix(conf.Path, "/") + "/development"
	case "prod":
		workdir = strings.TrimSuffix(conf.Path, "/") + "/production"
	default:
	}
	releaseDir := fmt.Sprintf("%s/releases", workdir)
	sharedDir := fmt.Sprintf("%s/shared", workdir)
	currentDir := fmt.Sprintf("%s/current", workdir)
	tags := conf.Tags
	version := time.Now().Format("20060102150405")

	var deploy Deploy
	db.First(&deploy, Deploy{SystemId: id, Stage: stage, Enable: true})
	if deploy.Id <= 0 {
		deploy = Deploy{
			SystemId: id,
			Stage:    stage,
			Operator: string(username),
			Version:  version,
		}
		err := db.Save(&deploy).Error
		if err != nil {
			sendFailMsg(r, "创建部署记录出错."+err.Error(), nil)
			return
		}
	} else {
		version = deploy.Version
	}

	cmds := NewShellCommand()
	// 1.创建部署目录
	cmds.Mkdir(workdir).Mkdir(releaseDir).Mkdir(sharedDir)

	// 2.创建新版本目录
	versionDir := fmt.Sprintf("%s/%s", releaseDir, version)
	cmds.Mkdir(versionDir)

	// 3.checkout代码
	if strings.Contains(conf.Repo, ".git") {
		cmds.GitUpdate(currentDir, versionDir, conf.Repo)
	} else {
		cmds.SvnUpdate(currentDir, versionDir, conf.Repo, conf.UserName, conf.Password)
	}

	// 4.处理共享目录
	// update 不需要

	// 5.清理备份版本
	cmds.ClearBackup(releaseDir, conf.BackupNum)

	// 6.TODO:测试?

	// 7.执行部署上线前命令
	if strings.TrimSpace(conf.BeforeCmd) != "" {
		conf.BeforeCmd = strings.Replace(conf.BeforeCmd, "$path", versionDir, -1)
		conf.BeforeCmd = strings.Replace(conf.BeforeCmd, "$share", sharedDir, -1)
		conf.BeforeCmd = strings.Replace(conf.BeforeCmd, "$release", releaseDir, -1)
		cmds.Exec(conf.BeforeCmd, versionDir)
	}

	// 执行对应场景上线前命令
	if stage == "dev" && strings.TrimSpace(conf.DevBeforeCmd) != "" {
		conf.DevBeforeCmd = strings.Replace(conf.DevBeforeCmd, "$path", versionDir, -1)
		conf.DevBeforeCmd = strings.Replace(conf.DevBeforeCmd, "$share", sharedDir, -1)
		conf.DevBeforeCmd = strings.Replace(conf.DevBeforeCmd, "$release", releaseDir, -1)
		cmds.Exec(conf.DevBeforeCmd, versionDir)
	}
	if stage == "prod" && strings.TrimSpace(conf.ProdBeforeCmd) != "" {
		conf.ProdBeforeCmd = strings.Replace(conf.ProdBeforeCmd, "$path", versionDir, -1)
		conf.ProdBeforeCmd = strings.Replace(conf.ProdBeforeCmd, "$share", sharedDir, -1)
		conf.ProdBeforeCmd = strings.Replace(conf.ProdBeforeCmd, "$release", releaseDir, -1)
		cmds.Exec(conf.ProdBeforeCmd, versionDir)
	}

	// 8.把current软链接指向新版本
	cmds.Ln(versionDir, currentDir)

	// 11.执行部署后命令
	if strings.TrimSpace(conf.AfterCmd) != "" {
		conf.AfterCmd = strings.Replace(conf.AfterCmd, "$path", versionDir, -1)
		conf.AfterCmd = strings.Replace(conf.AfterCmd, "$share", sharedDir, -1)
		conf.AfterCmd = strings.Replace(conf.AfterCmd, "$release", releaseDir, -1)
		cmds.Exec(conf.AfterCmd, versionDir)
	}

	// 执行对应场景上线后命令
	if stage == "dev" && strings.TrimSpace(conf.DevAfterCmd) != "" {
		conf.DevAfterCmd = strings.Replace(conf.DevAfterCmd, "$path", versionDir, -1)
		conf.DevAfterCmd = strings.Replace(conf.DevAfterCmd, "$share", sharedDir, -1)
		conf.DevAfterCmd = strings.Replace(conf.DevAfterCmd, "$release", releaseDir, -1)
		cmds.Exec(conf.DevAfterCmd, versionDir)
	}
	if stage == "prod" && strings.TrimSpace(conf.ProdAfterCmd) != "" {
		conf.ProdAfterCmd = strings.Replace(conf.ProdAfterCmd, "$path", versionDir, -1)
		conf.ProdAfterCmd = strings.Replace(conf.ProdAfterCmd, "$share", sharedDir, -1)
		conf.ProdAfterCmd = strings.Replace(conf.ProdAfterCmd, "$release", releaseDir, -1)
		cmds.Exec(conf.ProdAfterCmd, versionDir)
	}

	// 取对应该tag的所有服务器
	servers := getTagServers(tags)

	session := NewShellSession(servers, *cmds, deploy.Id)
	mutex.Lock()
	sessions[id] = session
	mutex.Unlock()

	startTime := time.Now()
	session.ParallelRun()
	if !session.Success {
		deploy.Stage = stage
		deploy.Operator = string(username)
		deploy.Status = -1
		deploy.Output = session.Output()
		deploy.ElapsedTime = int(time.Now().Sub(startTime).Seconds())
		deploy.CreatedAt = time.Now()
		db.Save(&deploy)

		sendFailMsg(r, "部署失败.", session.Output())
		return
	}

	// 取commit日志
	revision, commitLog := getCommitLog(session, id, stage, conf, currentDir)
	deploy.Revision = revision
	deploy.CommitLog = commitLog

	// 保存
	deploy.Stage = stage
	deploy.Operator = string(username)
	deploy.Status = 1
	deploy.Output = session.Output()
	deploy.Enable = true
	deploy.ElapsedTime = int(time.Now().Sub(startTime).Seconds())
	deploy.CreatedAt = time.Now()
	db.Save(&deploy)
	sendSuccessMsg(r, nil)
	return

}

// 回滚部署
func ExecuteRollback(username string, params martini.Params, r render.Render) {
	deployId, _ := strconv.Atoi(params["id"])

	var deploy Deploy
	db.First(&deploy, deployId)

	if deploy.Id <= 0 {
		sendFailMsg(r, "该版本不存在.", nil)
		return
	}

	id := deploy.SystemId
	stage := deploy.Stage
	if isDeploying(id) {
		sendFailMsg(r, "另一部署进程正在进行中，请稍候.", nil)
		return
	}

	var conf SystemConfig
	db.First(&conf, id)
	workdir := conf.Path
	switch deploy.Stage {
	case "dev":
		workdir = strings.TrimSuffix(conf.Path, "/") + "/development"
	case "prod":
		workdir = strings.TrimSuffix(conf.Path, "/") + "/production"
	default:
	}
	releaseDir := fmt.Sprintf("%s/releases", workdir)
	versionDir := fmt.Sprintf("%s/%s", releaseDir, deploy.Version)
	currentDir := fmt.Sprintf("%s/current", workdir)
	tags := conf.Tags

	cmds := NewShellCommand()
	// 判断该版本目录是否存在，存在直接回滚
	cmds.ExistDir(versionDir).Rollback(versionDir, currentDir)

	// 取对应该tag的所有服务器
	servers := getTagServers(tags)

	session := NewShellSession(servers, *cmds, deployId)
	mutex.Lock()
	sessions[id] = session
	mutex.Unlock()

	session.ParallelRun()
	if !session.Success {
		sendFailMsg(r, "回滚失败.", session.Output())
		return
	}

	// 去掉旧的部署的启用状态
	db.Model(Deploy{}).Where(Deploy{SystemId: id, Stage: stage, Enable: true}).Update(map[string]interface{}{"enable": false})

	deploy.Enable = true
	db.Save(&deploy)
	sendSuccessMsg(r, session.Output())
	return

}

func DeployProgress(params martini.Params, r render.Render) {
	id, _ := strconv.Atoi(params["id"])

	var s *ShellSession
	var found bool
	mutex.Lock()
	s, found = sessions[id]
	mutex.Unlock()

	if found {
		data := map[string]interface{}{}
		isComplete := false
		if s.IsComplete {
			output, status := getDeployLog(s.DeployId)
			if status != 0 {
				isComplete = true
				data["output"] = output
				data["complete"] = true
				data["success"] = s.Success
			}
		}

		if !isComplete {
			data["output"] = s.Output()
			data["complete"] = false
			data["success"] = false
		}

		r.JSON(200, data)
	} else {
		data := map[string]interface{}{}
		data["output"] = ""
		data["complete"] = true
		data["success"] = false
		r.JSON(200, data)
	}
}

func ShowDeployLog(params martini.Params, r render.Render) {
	id, _ := strconv.Atoi(params["id"])

	data := map[string]interface{}{}
	data["output"], data["success"] = getDeployLog(id)
	r.JSON(200, data)

}

func getDeployLog(id int) (output string, status int) {
	var deploy Deploy
	db.First(&deploy, id)

	status = deploy.Status

	if deploy.CommitLog != "" {
		output += "<span class=\"tip\">[变更] 版本提交历史log：\n      " +
			strings.Replace(deploy.CommitLog, "\n", "\n      ", -1) +
			"</span>\n"
	}
	output = output + deploy.Output

	return
}

func CancelDeploy(params martini.Params, r render.Render) {
	id, _ := strconv.Atoi(params["id"])

	mutex.Lock()
	defer mutex.Unlock()

	var deploy Deploy
	err := db.First(&deploy, id).Error
	if err != nil {
		sendFailMsg(r, "取消部署失败."+err.Error(), "")
		return
	}

	deploy.Status = -1
	db.Save(&deploy)

	s, found := sessions[deploy.SystemId]
	if found && !s.IsComplete {
		sessions[deploy.SystemId].Cancel()
		sessions[deploy.SystemId].Success = false
		sessions[deploy.SystemId].IsComplete = true
	}
	sendSuccessMsg(r, "取消部署成功.")
	return

}

func getCommitLog(session *ShellSession, id int, stage string, conf SystemConfig, currentDir string) (revision string, commitLogStr string) {
	var oldDeploy Deploy
	db.First(&oldDeploy, Deploy{SystemId: id, Stage: stage, Enable: true})

	var commitLog CommitLog
	if strings.Contains(conf.Repo, ".git") {
		commitLog, _ = session.RetrieveGitCommitLog(currentDir)
		session.IsComplete = true
	} else {
		commitLog, _ = session.RetrieveSvnCommitLog(currentDir, conf.UserName, conf.Password)
		session.IsComplete = true
	}
	if commitLog.LogEntries != nil {
		for _, commit := range commitLog.LogEntries {
			if commit.Revision == oldDeploy.Revision {
				break
			}

			commitLogStr += "r" + commit.Revision + "    " + commit.Msg + "\n"
		}
		if len(commitLog.LogEntries) > 0 {
			revision = commitLog.LogEntries[0].Revision
			if commitLogStr == "" {
				commitLogStr = "r" + commitLog.LogEntries[0].Revision + "  " + commitLog.LogEntries[0].Msg + "\n"
			}
		}
	}

	return
}

func isDeploying(systemid int) bool {
	mutex.Lock()
	defer mutex.Unlock()

	s, found := sessions[systemid]
	if !found || s.IsComplete {
		s = &ShellSession{}
		sessions[systemid] = s
		return false
	}

	return true
}
