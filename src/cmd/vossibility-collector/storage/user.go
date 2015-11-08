package storage

import (
	"strings"

	"github.com/mattbaird/elastigo/core"
)

const (
	UserIndex = "users"
	UserType  = "user"
)

type UserData struct {
	Login        string `json:"login"`
	Company      string `json:"company"`
	IsMaintainer bool   `json:"is_maintainer" toml:"is_maintainer"`
}

type userStore struct {
}

func (u *userStore) Get(login string) (*UserData, error) {
	var userData UserData
	if err := core.GetSource(UserIndex, UserType, strings.ToLower(login), map[string]interface{}{}, &userData); err != nil {
		return nil, err
	}
	userData.Login = login
	return &userData, nil
}
