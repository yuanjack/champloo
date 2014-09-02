package main

import (
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/binding"
	"github.com/martini-contrib/render"
)

type ActionMessage struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

var mutex *sync.Mutex = &sync.Mutex{}
var sessions map[int]*ShellSession = map[int]*ShellSession{}
var debug = flag.Bool("debug", false, "进入调试模式")

func main() {
	flag.Parse()

	err := InitDb()
	if err != nil {
		fmt.Printf("初始化sqllite db出错.%v\n", err)
		return
	}

	m := martini.Classic()
	//m.Use(martini.Static("public", martini.StaticOptions{Prefix: "/public"}))
	m.Use(BasicFunc(func(username string, password string) bool {
		var user User
		db.First(&user, User{Name: username})
		if user.Id <= 0 {
			return false
		}

		return username == user.Name && password == user.Password
	}))
	m.Use(render.Renderer(render.Options{
		Layout: "layout",
		Funcs: []template.FuncMap{
			{
				"formatTime": func(args ...interface{}) string {
					dt := time.Now().Sub(args[0].(time.Time).Local())
					if dt.Seconds() < 60 {
						return fmt.Sprintf("%d秒前", int(dt.Seconds()))
					}
					if dt.Minutes() < 60 {
						return fmt.Sprintf("%d分钟前", int(dt.Minutes()))
					}
					if dt.Hours() < 24 {
						return fmt.Sprintf("%d小时前", int(dt.Hours()))
					}
					return args[0].(time.Time).Local().Format("2006/01/02 15:04:05")
				},
				"unescaped": func(args ...interface{}) template.HTML {
					return template.HTML(args[0].(string))
				},
				"containtag": func(args ...interface{}) bool {
					return strings.Contains(","+args[0].(string)+",", ","+args[1].(string)+",")
				},
				"getServerStatusClass": func(args ...interface{}) template.HTML {
					if args[1].(bool) {
						return "active"
					}
					if time.Now().Sub(args[0].(time.Time).Local()).Minutes() > 5 {
						return "danger"
					}
					return ""
				},
			},
		},
	}))

	m.Get("/", func(username AuthUser, r render.Render) {
		var confs []SystemConfig
		db.Order("id desc").Find(&confs)

		for i := 0; i < len(confs); i++ {
			var deploy Deploy
			db.Select("id, version, operator, status, created_at").First(&deploy, Deploy{SystemId: confs[i].Id, Enable: true})
			confs[i].EnableDeploy = deploy
		}

		data := map[string]interface{}{"username": username, "confs": confs}
		r.HTML(200, "index", data)
	})
	m.Get("/users", func(username AuthUser, r render.Render) {
		var users []User
		db.Order("id desc").Find(&users)

		data := map[string]interface{}{"username": username, "users": users}
		r.HTML(200, "user", data)
	})
	m.Post("/users", binding.Bind(User{}), func(user User, r render.Render) {
		user.CreatedAt = time.Now()
		err := db.Save(&user).Error
		if user.Id > 0 {
			sendSuccessMsg(r, "")
		} else {
			sendFailMsg(r, "保存失败."+err.Error(), "")
		}
	})
	m.Get("/build/:id", func(username AuthUser, params martini.Params, r render.Render) {
		id, _ := strconv.Atoi(params["id"])

		var conf SystemConfig
		db.First(&conf, id)

		var deploys []Deploy
		db.Limit(10).Order("id desc").Find(&deploys, Deploy{SystemId: id})

		data := map[string]interface{}{"username": username, "conf": conf, "deploys": deploys}
		r.HTML(200, "build", data)
	})
	m.Post("/deploy/:id", ExecuteDeployDefault)
	m.Post("/deploy/dev/:id", ExecuteDeployDev)
	m.Post("/deploy/prod/:id", ExecuteDeployProd)
	m.Get("/deploy/:id/progress", func(params martini.Params, r render.Render) {
		id, _ := strconv.Atoi(params["id"])

		var s *ShellSession
		var found bool
		mutex.Lock()
		s, found = sessions[id]
		mutex.Unlock()

		if found {
			data := map[string]interface{}{}
			data["output"] = s.Output()
			data["complete"] = s.IsComplete
			r.JSON(200, data)
		} else {
			data := map[string]interface{}{}
			data["output"] = ""
			data["complete"] = true
			r.JSON(200, data)
		}
	})
	m.Get("/deploy/:id/log", func(params martini.Params, r render.Render) {
		id, _ := strconv.Atoi(params["id"])

		var deploy Deploy
		db.First(&deploy, id)

		data := map[string]interface{}{}
		data["output"] = deploy.Output
		r.JSON(200, data)

	})
	m.Post("/deploy/:id/rollback", ExecuteRollback)
	m.Get("/config", func(username AuthUser, r render.Render) {
		var servers []Server
		db.Select("tags").Find(&servers)

		tagsMap := map[string]bool{}
		tagsMap["全部"] = true
		for _, server := range servers {
			if server.Tags == "" {
				continue
			}

			arr := strings.Split(server.Tags, ",")
			for _, tag := range arr {
				if strings.TrimSpace(tag) == "" {
					continue
				}

				tagsMap[strings.TrimSpace(tag)] = true
			}
		}

		tags := []string{}

		for key, _ := range tagsMap {
			tags = append(tags, key)
		}

		data := map[string]interface{}{"username": username, "tags": tags, "conf": SystemConfig{Way: "checkout", BackupNum: 10}}
		r.HTML(200, "config", data)
	})
	m.Post("/config", func(req *http.Request, params martini.Params, r render.Render) {
		req.ParseForm()
		id, _ := strconv.Atoi(req.PostForm.Get("id"))
		name := req.PostForm.Get("name")
		enbaleDevStage := req.PostForm.Get("dev-stage") == "on"
		enbaleProdStage := req.PostForm.Get("prod-stage") == "on"
		way := req.PostForm.Get("way")
		path := req.PostForm.Get("path")
		shared := req.PostForm.Get("shared")
		num, _ := strconv.Atoi(req.PostForm.Get("num"))
		repo := req.PostForm.Get("repo")
		username := req.PostForm.Get("username")
		password := req.PostForm.Get("password")
		beforecmd := req.PostForm.Get("before-cmd")
		aftercmd := req.PostForm.Get("after-cmd")
		devBeforeCmd := req.PostForm.Get("dev-before-cmd")
		devAfterCmd := req.PostForm.Get("dev-after-cmd")
		prodBeforeCmd := req.PostForm.Get("prod-before-cmd")
		prodAfterCmd := req.PostForm.Get("prod-after-cmd")
		tagsSlice := req.Form["tags"]
		tags := ""
		if len(tagsSlice) > 0 {
			tags = strings.Join(tagsSlice, ",")
		}

		conf := SystemConfig{
			Id:              id,
			Name:            name,
			EnableDevStage:  enbaleDevStage,
			EnableProdStage: enbaleProdStage,
			Way:             way,
			Path:            path,
			Shared:          shared,
			BackupNum:       num,
			Repo:            repo,
			UserName:        username,
			Password:        password,
			Tags:            tags,
			BeforeCmd:       beforecmd,
			AfterCmd:        aftercmd,
			DevBeforeCmd:    devBeforeCmd,
			DevAfterCmd:     devAfterCmd,
			ProdBeforeCmd:   prodBeforeCmd,
			ProdAfterCmd:    prodAfterCmd,
		}
		err = db.Save(&conf).Error
		if err != nil {
			r.JSON(200, ActionMessage{
				Success: false,
				Message: "更新出错." + err.Error(),
			})
			return
		}
		r.JSON(200, ActionMessage{
			Success: true,
			Message: "成功",
			Data:    conf,
		})
	})
	m.Get("/config/:id", func(username AuthUser, params martini.Params, r render.Render) {
		id, _ := strconv.Atoi(params["id"])

		var servers []Server
		db.Select("tags").Find(&servers)

		tagsMap := map[string]bool{}
		tagsMap["全部"] = true
		for _, server := range servers {
			if server.Tags == "" {
				continue
			}

			arr := strings.Split(server.Tags, ",")
			for _, tag := range arr {
				if strings.TrimSpace(tag) == "" {
					continue
				}

				tagsMap[strings.TrimSpace(tag)] = true
			}
		}

		tags := []string{}

		for key, _ := range tagsMap {
			tags = append(tags, key)
		}

		var conf SystemConfig
		db.First(&conf, id)
		data := map[string]interface{}{"username": username, "tags": tags, "conf": conf}
		r.HTML(200, "config", data)
	})
	m.Get("/servers", GetServers)
	m.Delete("/servers/:id", DeleteServer)
	m.Put("/servers/:id", binding.Bind(Server{}), EditServer)
	m.Put("/servers/:id/toggle", ToggleServer)

	m.Get("/api/heartbeat", Heartbeat)
	m.Get("/avatar/.*", GenAvatar)
	m.Run()
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

func sendSuccessMsg(r render.Render, data interface{}) {
	r.JSON(200, ActionMessage{
		Success: true,
		Message: "成功",
		Data:    data,
	})
}

func sendFailMsg(r render.Render, msg string, data interface{}) {
	r.JSON(200, ActionMessage{
		Success: false,
		Message: msg,
		Data:    data,
	})
}

func getTagServers(tags string) []Server {
	var servers []Server
	db.Find(&servers)
	tagServers := []Server{}

	arr := strings.Split(tags, ",")
	for _, srv := range servers {
		for _, tag := range arr {
			if strings.Contains(srv.Tags+",", tag+",") {
				tagServers = append(tagServers, srv)
				continue
			}
		}
	}

	return tagServers
}
