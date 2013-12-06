// Copyright 2013 Denis Pobedrya <denis.pobedrya@gmail.com> All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"github.com/thoj/go-ircevent"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	Goal int
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
	goal := 0
	if u, ok := users[user]; ok {
		goal = u.Goal
	}
	users[user] = UserData{date, goal}
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

func daystr(days int) string {
	if days == 1 {
		return " day"
	} else {
		return " days"
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
			if user == nick {
				return "Your counter updated"
			} else {
				return "Counter of user " + user + " updated"
			}
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
		start := user + " is winning for "
		if user == "" {
			user = nick
			start = "You're winning for "
		}
		if u, ok := users[user]; ok {
			dur := time.Now().Sub(u.Date)
			days := int(dur.Hours() / 24)
			dstr := daystr(days)
			ret := start + strconv.Itoa(days) + dstr
			if u.Goal > days && u.Goal > 0 {
				left := u.Goal - days
				ret = ret + ". You have " + strconv.Itoa(left) + daystr(left) + " to go for your goal."
			} else if u.Goal == days {
				ret = ret + ". Congratulations on reaching your goal, feel free to set a new one."
			}
			return ret
		}
		if user == nick {
			return "You don't have a counter, use \"set yyyy-mm-dd\" or \"reset\" to get one"
		}
		return "Counter not found for user " + user
	},
	"help": func(nick, data string) string {
		return "Commands: help, set, get, reset, delete, hello"
	},
	"delete": func(nick, data string) string {
		user := data
		if user == "" {
			user = nick
		}
		if user == nick || isAdmin(nick) {
			if _, ok := users[user]; ok {
				delete(users, user)
				save()
				if user == nick {
					return "Your counter deleted"
				} else {
					return "Counter of user " + user + " deleted"
				}
			} else {
				return "Counter not found for user " + user
			}
		}
		return "You can't delete other people's counters"
	},
	"setgoal": func(nick, data string) string {
		if u, ok := users[nick]; ok {
			goal, err := strconv.ParseInt(data, 10, 32)
			if err != nil {
				return "Error setting goal, use \"setgoal <number of days>\""
			}
			u.Goal = int(goal)
			users[nick] = u
			save()
			return "Your goal has been set"
		}
		return "You don't have a counter, use \"set yyyy-mm-dd\" or \"reset\" to get one"
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
		cmd, data := strings.ToLower(m[1]), m[2]
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
	baseReStr := "(\\w+)(?:\\s(.+))?"
	cmdReStr := "(?i)" + baseReStr //regex for command in private
	chatCmdReStr := "(?i)^" + nick + "[:,]?\\s" + cmdReStr
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
