package rasant

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/shaynemeyer/rasant/render"
)

const version = "1.0.0"

type Rasant struct {
	AppName string
	Debug bool
	Version string
	ErrorLog *log.Logger
	InfoLog *log.Logger
	RootPath string
	Routes *chi.Mux
	Render *render.Render
	config config
}

type config struct {
	port string
	renderer string
}

func (ras *Rasant) New(rootPath string) error {
	pathConfig := initPaths{
		rootPath: rootPath,
		folderNames: []string{"handlers", "migrations", "views", "data", "public", "tmp", "logs", "middleware"},
	}

	err := ras.Init(pathConfig)
	if err!= nil {
		return err
	}

	err = ras.checkDotEnv(rootPath) 
	if err!= nil {
		return err
	}

	// read .env
	err = godotenv.Load(rootPath + "/.env")
	if err!= nil {
    return err
  }

	// create loggers
	infoLog, errorLog := ras.startLoggers()
	ras.InfoLog = infoLog
	ras.ErrorLog = errorLog
	ras.Debug, _ = strconv.ParseBool(os.Getenv("DEBUG"))
	ras.Version = version
	ras.RootPath = rootPath
	ras.Routes = ras.routes().(*chi.Mux)
	
	ras.config = config{
		port: os.Getenv("PORT"),
    renderer: os.Getenv("RENDERER"),
	}

	ras.createRenderer()

	return nil
}

func (ras *Rasant) Init(p initPaths) error {
	root := p.rootPath
	for _, path := range p.folderNames {
		// create directory if it doesn't exist
		err := ras.CreateDirIfNotExist(root + "/" + path)
		if err!= nil {
      return err
    }
	}
	return nil
}

// ListenAndServe starts the web server
func (ras *Rasant) ListenAndServe() {
	srv := &http.Server{
		Addr: fmt.Sprintf(":%s", os.Getenv("PORT")),
		ErrorLog: ras.ErrorLog,
		Handler: ras.Routes,
		IdleTimeout: 30 * time.Second,
		ReadTimeout: 30 * time.Second,
		WriteTimeout: 600 * time.Second,
	}

	ras.InfoLog.Printf("Listing on port %s", os.Getenv("PORT"))
	err := srv.ListenAndServe()
	ras.ErrorLog.Fatal(err)
}

func (ras *Rasant) checkDotEnv(path string) error { 
	err := ras.CreateFileIfNotExist(fmt.Sprintf("%s/.env", path))
	if err!= nil {
		return err
	}
	return nil
}

func (ras *Rasant) startLoggers() (*log.Logger, *log.Logger) {
	var infoLog *log.Logger
	var errorLog *log.Logger

	infoLog = log.New(os.Stdout, "INFO\t", log.Ldate| log.Ltime)
	errorLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	return infoLog, errorLog
}

func (ras *Rasant) createRenderer() {
	myRenderer := render.Render{
		Renderer: ras.config.renderer,
		RootPath: ras.RootPath,
		Port: ras.config.port,
	}

	ras.Render = &myRenderer
}
