package rasant

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/shaynemeyer/rasant/render"
	"github.com/shaynemeyer/rasant/session"
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
	Session *scs.SessionManager
	DB Database
	JetViews *jet.Set
	config config
}

type config struct {
	port string
	renderer string
	cookie cookieConfig
	sessionType string
	database databaseConfig
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

	// connect to database
	if os.Getenv("DATABASE_TYPE") != "" {
		db, err := ras.OpenDB(os.Getenv("DATABASE_TYPE"), ras.BuildDSN())
		if err!= nil {
      errorLog.Println(err)
			os.Exit(1)
    }
		ras.DB = Database{
			DataType: os.Getenv("DATABASE_TYPE"),
			Pool: db,
		}
	}

	ras.InfoLog = infoLog
	ras.ErrorLog = errorLog
	ras.Debug, _ = strconv.ParseBool(os.Getenv("DEBUG"))
	ras.Version = version
	ras.RootPath = rootPath
	ras.Routes = ras.routes().(*chi.Mux)
	
	ras.config = config{
		port: os.Getenv("PORT"),
    renderer: os.Getenv("RENDERER"),
		cookie: cookieConfig{
			name: os.Getenv("COOKIE_NAME"),
			lifetime: os.Getenv("COOKIE_LIFETIME"),
			persist: os.Getenv("COOKIE_PERSISTS"),
			secure: os.Getenv("COOKIE_SECURE"),
			domain: os.Getenv("COOKIE_DOMAIN"),
		},
		sessionType: os.Getenv("SESSION_TYPE"),
		database: databaseConfig{
			database: os.Getenv("DATABASE_TYPE"),
			dsn: ras.BuildDSN(),
		},
	}

	// create session
	sess := session.Session {
		CookieLifetime: ras.config.cookie.lifetime,
		CookiePersist: ras.config.cookie.persist,
		CookieName: ras.config.cookie.name,
		SessionType: ras.config.sessionType,
		CookieDomain: ras.config.cookie.domain,
	}

	ras.Session = sess.InitSession()

	var views = jet.NewSet(
		jet.NewOSFileSystemLoader(fmt.Sprintf("%s/views", rootPath)),
		jet.InDevelopmentMode(),
	)

	ras.JetViews = views

	ras.createRenderer()

	return nil
}

// Init creates necessary folders for our Celeritas application
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

	defer ras.DB.Pool.Close()

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
		JetViews: ras.JetViews,
	}

	ras.Render = &myRenderer
}

func (ras *Rasant) BuildDSN() string {
	var dsn string

	switch os.Getenv("DATABASE_TYPE"){
	case "postgres", "postgresql":
		dsn = fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s timezone=UTC connect_timeout=5", 
		os.Getenv("DATABASE_HOST"), 
		os.Getenv("DATABASE_PORT"), 
		os.Getenv("DATABASE_USER"), 
		os.Getenv("DATABASE_NAME"),
		os.Getenv("DATABASE_SSL_MODE"))
		
		// we check to see if a database passsword has been supplied, since including "password=" with nothing
		// after it sometimes causes postgres to fail to allow a connection.
		if os.Getenv("DATABASE_PASS") != "" {
			dsn = fmt.Sprintf("%s password=%s", dsn, os.Getenv("DATABASE_PASS"))
		}
	default:

	}

	return dsn
}