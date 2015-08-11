package main

import (
	"fmt"

	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/mattbaird/elastigo/core"
)

var updateUsersCommand = cli.Command{
	Name:   "update_users",
	Usage:  "update the user store with the information from a file",
	Action: doUpdateUsers,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "f, file",
			Value: "users.toml",
			Usage: "users description file",
		},
	},
}

// doUpdateUsers synchronize the content of the specified user file with the
// Elastic Search backend.
func doUpdateUsers(c *cli.Context) {
	_ = ParseConfigOrDie(c.GlobalString("config"))

	var userData map[string]UserData
	if _, err := toml.DecodeFile(c.String("file"), &userData); err != nil {
		log.Fatal(err)
	}

	for login, data := range userData {
		fmt.Printf("Saving data for %q: %#v\n", login, data)
		if _, err := core.Index(UserIndex, UserType, login, nil, data); err != nil {
			log.Errorf("indexing data for %q; %v", login, err)
		}
	}
}
