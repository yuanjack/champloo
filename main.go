package main

import (
	"flag"
	"fmt"
	"html/template"
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

		var user User
		db.First(&user, User{Name: string(username)})

		for i := 0; i < len(confs); i++ {
			var deploy Deploy
			db.Order("id desc").Select("id, version, operator, status, created_at").First(&deploy, Deploy{SystemId: confs[i].Id, Enable: true})
			confs[i].EnableDeploy = deploy

			var star UserStar
			db.First(&star, UserStar{SystemId: confs[i].Id, UserId: user.Id})
			if star.Id > 0 {
				confs[i].IsUserStar = true
			}
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
	m.Get("/deploy/:id/progress", DeployProgress)
	m.Get("/deploy/:id/log", GetDeployLog)
	m.Post("/deploy/:id/cancel", CancelDeploy)
	m.Post("/deploy/:id/rollback", ExecuteRollback)
	m.Get("/config", NewSystem)
	m.Post("/config", SaveSystem)
	m.Get("/config/:id", GetSystemById)
	m.Put("/config/:id/star", ToggleStarSystem)
	m.Get("/servers", GetServers)
	m.Delete("/servers/:id", DeleteServer)
	m.Put("/servers/:id", binding.Bind(Server{}), EditServer)
	m.Put("/servers/:id/toggle", ToggleServer)

	m.Get("/api/heartbeat", Heartbeat)
	m.Get("/avatar/.*", GenAvatar)
	m.Run()
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
