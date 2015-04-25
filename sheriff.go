package main

import (
	"fmt"
	conf "github.com/KarinaMobileSolutions/config"
	"github.com/fzzy/radix/redis"
	"github.com/robfig/cron"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Conf struct {
	Scripts []Script `json:"scripts"`
	Redis   Database `json:"redis"`
}

type Script struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Directory   string   `json:"directory"`
	Format      string   `json:"format"`
	Cmd         string   `json:"cmd"`
	Args        []string `json:"args"`
	Status      Status   `json:"status"`
	StatusSort  string   `json:"status_sort"`
}

type Database struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int64  `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Status struct {
	Critical float64 `json:"critical"`
	Warning  float64 `json:"warning"`
	Ok       float64 `json:"ok"`
}

var (
	config Conf
)

func ErrorHandler(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
}

func StoreScripts(script Script) {
	RedisClient, err := redis.Dial(config.Redis.Type, config.Redis.Host+":"+strconv.FormatInt(config.Redis.Port, 10))
	ErrorHandler(err)

	defer RedisClient.Close()

	RedisClient.Cmd("sAdd", "sheriff:scripts", script.Name)
	RedisClient.Cmd("hSet", "sheriff:scripts:"+script.Name, "format", script.Format)
	RedisClient.Cmd("hSet", "sheriff:scripts:"+script.Name, "description", script.Description)
	RedisClient.Cmd("hSet", "sheriff:scripts:"+script.Name, "directory", script.Directory)
	RedisClient.Cmd("hSet", "sheriff:scripts:"+script.Name, "cmd", script.Cmd)
	RedisClient.Cmd("hSet", "sheriff:scripts:"+script.Name, "args", strings.Join(script.Args, " "))
	RedisClient.Cmd("hSet", "sheriff:scripts:"+script.Name, "status_sort", script.StatusSort)

	RedisClient.Cmd("hSet", "sheriff:scripts:"+script.Name+":status", "critical", script.Status.Critical)
	RedisClient.Cmd("hSet", "sheriff:scripts:"+script.Name+":status", "warning", script.Status.Warning)
	RedisClient.Cmd("hSet", "sheriff:scripts:"+script.Name+":status", "ok", script.Status.Ok)
}

func main() {
	conf.Init(&config)
	fmt.Println("Adding Scripts routines...")

	var wg sync.WaitGroup

	cr := cron.New()
	for _, script := range config.Scripts {

		StoreScripts(script)

		wg.Add(1)
		script := script
		cr.AddFunc(script.Format, func() {

			RedisClient, err := redis.Dial(config.Redis.Type, config.Redis.Host+":"+strconv.FormatInt(config.Redis.Port, 10))
			ErrorHandler(err)

			defer RedisClient.Close()

			cmd := exec.Command(script.Cmd, script.Args...)

			cmd.Dir = script.Directory

			output, err := cmd.Output()

			if err != nil {
				fmt.Printf("Error calling %s: %v\n", script.Cmd, err)
			} else {
				output := string(output[:])
				var value float64
				value, err := strconv.ParseFloat(output, 64)

				if err != nil {
					fmt.Printf("Error parsing value %s: %v\n", script.Cmd, err)
				} else {
					t := time.Now().Unix()
					RedisClient.Cmd("zAdd", "sheriff:"+script.Name, float64(t), strconv.FormatInt(t, 10)+":"+strconv.FormatFloat(value, 'f', -1, 64))
					RedisClient.Cmd("lPush", "sheriff:realtime", script.Name+":"+strconv.FormatInt(t, 10)+":"+strconv.FormatFloat(value, 'f', -1, 64))
				}
			}
		})

		fmt.Printf("Script %v added\n", script.Name)
	}

	cr.Start()

	wg.Wait()
}
