Description
===========
This is simple IRC bot for managing day counters.

Compiling
=========
    go build CounterBot.go

Usage
=====
Run CounterBot binary in same directory as conf.json (example given in conf_example.json)
It will also create users.json for tracking dates

Bot commands
==========
* hello # test command
* set &lt;yyyy-mm-dd> [user] # set counter date
* reset # set counter date to today
* get [user] # get day count
* help # list of commands
