package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
)

func setup(arg1, arg2 string) {
	if arg1 != "new" && arg1 != "version" && arg1 != "help" {
		err := godotenv.Load()
		if err!= nil {
			exitGracefully(err)
		}

		path, err := os.Getwd()
		if err != nil {
			exitGracefully(err)
		}

		ras.RootPath = path
		ras.DB.DataType = os.Getenv("DATABASE_TYPE")
	}
}

func getDSN() string {
	dbType := ras.DB.DataType

	if dbType == "pgx" {
		dbType = "postgres"
	}

	if dbType == "postgres" {
		var dsn string
		if os.Getenv("DATABASE_PASS") != "" {
			dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
        os.Getenv("DATABASE_USER"),
        os.Getenv("DATABASE_PASS"),
        os.Getenv("DATABASE_HOST"),
        os.Getenv("DATABASE_PORT"),
				os.Getenv("DATABASE_NAME"),
				os.Getenv("DATABASE_SSL_MODE"))
		} else {
			dsn = fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=%s",
        os.Getenv("DATABASE_USER"),
        os.Getenv("DATABASE_HOST"),
        os.Getenv("DATABASE_PORT"),
				os.Getenv("DATABASE_NAME"),
				os.Getenv("DATABASE_SSL_MODE"))
		}
		return dsn
	} 
		
	return "mysql://" + ras.BuildDSN()
}

func showHelp() {
	color.Yellow(`Available commands:

	help                  - show the help commands
	version               - print application version
	migrate               - runs all up migrations that have not run previously
	migrate down          - reverses the most recent migration
	migrate reset         - runs all down migrations in reverse order, and then all up migrations
	make migration <name> - creates two new up and down migrations in the migrations folder
	make auth             - creates and runs migrations for authentication tables, and creates models and middleware
	make handler <name>   - creates a stub handler in the handlers directory
	make model <name>     - creates a new model in the data directory
	make session          - creates a table in the database as a session store
	`)
}

func updateSourceFiles(path string, fi os.FileInfo, err error) error {
	// check for an error
	if err != nil {
		return err
	}

	// check if current file is directory
	if fi.IsDir() {
		return nil
	}

	// only check go files
	matched, err := filepath.Match("*.go", fi.Name())
	if err!= nil {
    return err
  }

	// we have a matching file
	if matched {
    read, err := os.ReadFile(path)
    if err!= nil {
      exitGracefully(err)
    }

		newContents := strings.Replace(string(read), "myapp", appURL, -1)

		// write the changed file
		err = os.WriteFile(path, []byte(newContents), 0)
    if err!= nil {
      exitGracefully(err)
    }
  }

	return nil
}

func updateSource() {
	// walk entire project directory, including subdirectories
	err := filepath.Walk(".", updateSourceFiles)
	if err!= nil {
		exitGracefully(err)
	}
}