package main

import (
	"fmt"
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
	Scripts   []Script   `json:"scripts"`
	Databases []Database `json:"dbs"`
}

type Script struct {
	Name      string   `json:"name"`
	Directory string   `json:"directory"`
	Format    string   `json:"format"`
	Cmd       string   `json:"cmd"`
	Args      []string `json:"args"`
}
type Database struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

var (
	config   Conf
	rediscon Database
)

func ErrorHandler(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
}

func StoreScripts(script Script, RedisClient *redis.Client) {
	RedisClient.Cmd("sAdd", "sheriff:scripts", script.Name)
	RedisClient.Cmd("hSet", "sheriff:scripts:"+script.Name, "format", script.Format)
	RedisClient.Cmd("hSet", "sheriff:scripts:"+script.Name, "directory", script.Directory)
	RedisClient.Cmd("hSet", "sheriff:scripts:"+script.Name, "cmd", script.Cmd)
	RedisClient.Cmd("hSet", "sheriff:scripts:"+script.Name, "args", strings.Join(script.Args, " "))
}

func main() {
	fmt.Println("Connecting to redis server...")
	RedisClient, err := redis.Dial("tcp", "127.0.0.1:6379")
	ErrorHandler(err)

	defer RedisClient.Close()

	ParseScripts()
	fmt.Println("Adding Scripts routines...")

	var wg sync.WaitGroup

	cr := cron.New()
	for _, script := range config.Scripts {

		StoreScripts(script, RedisClient)

		wg.Add(1)
		script := script
		cr.AddFunc(script.Format, func() {

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
