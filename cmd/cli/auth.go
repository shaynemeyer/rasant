package main

import (
	"fmt"
	"time"
)

func doAuth() error {
	// migrations
	dbType := ras.DB.DataType
	fileName := fmt.Sprintf("%d_create_auth_tables", time.Now().UnixMicro())
	upFile := ras.RootPath + "/migrations/" + fileName + ".up.sql"
	downFile := ras.RootPath + "/migrations/" + fileName + ".down.sql"

	err := copyFileFromTemplate("templates/migrations/auth_tables." + dbType + ".sql", upFile)
	if err != nil {
		exitGracefully(err)
	}

	err = copyDataToFile([]byte("drop table if exists users cascade; drop table if exists tokens cascade; drop table if exists remember_tokens;"), downFile)
	if err != nil {
		exitGracefully(err)
	}

	// run migrations
	err = doMigrate("up", "")
	if err != nil {
		exitGracefully(err)
	}

	// copy files over
	err = copyFileFromTemplate("templates/data/user.go.txt", ras.RootPath + "/data/user.go")
	if err != nil {
		exitGracefully(err)
	}

	err = copyFileFromTemplate("templates/data/token.go.txt", ras.RootPath + "/data/token.go")
	if err != nil {
		exitGracefully(err)
	}

	// copy over middleware
	err = copyFileFromTemplate("templates/middleware/auth.go.txt", ras.RootPath + "/middleware/auth.go")
	if err != nil {
		exitGracefully(err)
	}

	err = copyFileFromTemplate("templates/middleware/auth-token.go.txt", ras.RootPath + "/middleware/auth-token.go")
	if err != nil {
		exitGracefully(err)
	}

	return nil
}