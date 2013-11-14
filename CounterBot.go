package main

import (
	"encoding/json"
	"fmt"
	"github.com/thoj/go-ircevent"
	"io/ioutil"
	"regexp"
	"time"
	"strconv"
	"os"
)

type Config struct {
	Nick     string
	User     string
	Server   string
	Channels []string
	Admins   []string
}

type UserData struct {
	Date time.Time
}

var (
	users     map[string]UserData
	admins    []string
	cmdRe     *regexp.Regexp
	chatCmdRe *regexp.Regexp
	dateRe    *regexp.Regexp
)

func loadConfig(path string) Config {
	var c Config
	b, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(b, &c)
	if err != nil {
		panic(err)
	}
	return c
}

func isAdmin(nick string) bool {
	for _, s := range admins {
		if nick == s {
			return true
		}
	}
	return false
}

func setDate(user string, date time.Time) {
	users[user] = UserData{date}
	save()
}

func save() {
	data, _ := json.Marshal(users)
	ioutil.WriteFile("users.json", data, os.FileMode(0644))
}

func load() {
	b, err := ioutil.ReadFile("users.json")
	if err != nil {
		fmt.Println("Error reading users file")
		return
	}
	err = json.Unmarshal(b, &users)
	if err != nil {
		fmt.Println("Error parsing users file")
	}
}

var funcs = map[string]func(string, string) string{
	"hello": func(nick, data string) string {
		return "Hello, " + nick
	},
	"set": func(nick, data string) string {
		p := dateRe.FindStringSubmatch(data)
		if p == nil {
			return "Data format is yyyy-mm-dd"
		}
		dateStr := p[1]
		user := p[2]
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return err.Error()
		}
		if user == "" {
			user = nick
		}
		if user == nick || isAdmin(nick) {
			setDate(user, date)
			return "Counter of user " + user + " updated"
		} else {
			return "Sorry, you can't change other people's counters"
		}
	},
	"reset": func(nick, data string) string {
		user := data
		if user == "" {
			user = nick
		}
		now := time.Now()
		if user == nick || isAdmin(nick) {
			setDate(user, time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC))
			return "Counter for user " + user + " updated"
		}
		return "Sorry, you can't change other people's counters"
	},
	"get": func(nick, data string) string {
		user := data
		if user == "" {
			user = nick
		}
		if u, ok := users[user]; ok {
			dur := time.Now().Sub(u.Date)
			days := int(dur.Hours() / 24)
			var dstr string
			if days == 1 {
				dstr = " day"
			} else {
				dstr = " days"
			}
			return user + " is winning for " + strconv.Itoa(days) + dstr
		}
		return "Counter not found for user " + user
	},
	"help": func(nick, data string) string {
		return "Commands: help, set, get, reset, hello"
	},
}

func onMessage(bot *irc.Connection, from, to, message string) {
	fmt.Printf("%s => %s: %s\n", from, to, message)
	re := cmdRe
	answer := ""
	answerTo := from
	if to[0] == '#' {
		answer = from + ": "
		re = chatCmdRe
		answerTo = to
	}
	m := re.FindStringSubmatch(message)
	if m != nil {
		cmd, data := m[1], m[2]
		fmt.Println("Cmd:", cmd, "Data:", data)
		if funcs[cmd] != nil {
			answer += funcs[cmd](from, data)
		} else {
			answer += "Command not found"
		}
		bot.Privmsg(answerTo, answer)
	}
}

func compileRegex(nick string) {
	cmdReStr := "(\\w+)(?:\\s(.+))?" //regex for command in private
	chatCmdReStr := "^" + nick + "[:,]\\s" + cmdReStr
	cmdRe, _ = regexp.Compile(cmdReStr)
	chatCmdRe, _ = regexp.Compile(chatCmdReStr)
	dateRe, _ = regexp.Compile(`(\d\d\d\d-\d?\d-\d?\d)(?:\s(\w+))?`)
}

func main() {
	config := loadConfig("conf.json")
	admins = config.Admins
	users = make(map[string]UserData)
	load()
	compileRegex(config.Nick)
	bot := irc.IRC(config.Nick, config.User)
	bot.Connect(config.Server)
	for _, ch := range config.Channels {
		bot.Join(ch)
	}
	bot.AddCallback("PRIVMSG", func(e *irc.Event) {
		onMessage(bot, e.Nick, e.Arguments[0], e.Message)
	})
	bot.Loop()
}
