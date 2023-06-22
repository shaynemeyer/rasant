package rasant

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/alexedwards/scs/v2"
	"github.com/dgraph-io/badger/v3"
	"github.com/go-chi/chi/v5"
	"github.com/gomodule/redigo/redis"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	"github.com/shaynemeyer/rasant/cache"
	"github.com/shaynemeyer/rasant/mailer"
	"github.com/shaynemeyer/rasant/render"
	"github.com/shaynemeyer/rasant/session"
)

const version = "1.0.0"

var myRedisCache *cache.RedisCache
var myBadgerCache *cache.BadgerCache
var redisPool *redis.Pool
var badgerConn *badger.DB

// Rasant is the overall type for the Rasant package. Members that are exported in this type
// are available to any application that uses it.
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
	EncryptionKey string
	Cache cache.Cache
	Scheduler *cron.Cron
	Mail mailer.Mail
	Server Server
}

type Server struct {
	ServerName string
	Port string
	Secure bool
	URL string
}

type config struct {
	port string
	renderer string
	cookie cookieConfig
	sessionType string
	database databaseConfig
	redis redisConfig
}

// New reads the .env file, creates our application config, populates the Rasant type with settings
// based on .env values, and creates necessary folders and files if they don't exist
func (ras *Rasant) New(rootPath string) error {
	pathConfig := initPaths{
		rootPath: rootPath,
		folderNames: []string{"handlers", "migrations", "views", "mail", "data", "public", "tmp", "logs", "middleware"},
	}

	err := ras.Init(pathConfig)
	if err != nil {
		return err
	}

	err = ras.checkDotEnv(rootPath) 
	if err != nil {
		return err
	}

	// read .env
	err = godotenv.Load(rootPath + "/.env")
	if err != nil {
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

	scheduler := cron.New()
	ras.Scheduler = scheduler

	if os.Getenv("CACHE") == "redis" || os.Getenv("SESSION_TYPE") == "redis" {
		myRedisCache = ras.createClientRedisCache()
		ras.Cache = myRedisCache
		redisPool = myRedisCache.Conn
	}

	if os.Getenv("CACHE") == "badger" {
		myBadgerCache = ras.createClientBadgerCache()
		ras.Cache = myBadgerCache
		badgerConn = myBadgerCache.Conn

		_, err = ras.Scheduler.AddFunc("@daily", func() {
			_ = myBadgerCache.Conn.RunValueLogGC(0.7)
		})

		if err != nil {	
			return err
		}
	}

	ras.InfoLog = infoLog
	ras.ErrorLog = errorLog
	ras.Debug, _ = strconv.ParseBool(os.Getenv("DEBUG"))
	ras.Version = version
	ras.RootPath = rootPath
	ras.Mail = ras.createMailer()
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
		redis: redisConfig{
			host: os.Getenv("REDIS_HOST"),
			password: os.Getenv("REDIS_PASSWORD"),
			prefix: os.Getenv("REDIS_PREFIX"),
		},
	}

	secure := true 
	if strings.ToLower(os.Getenv("SECURE")) == "false" {
		secure = false
	}

	ras.Server = Server{
		ServerName: os.Getenv("SERVER_NAME"),
		Port: os.Getenv("PORT"),
    Secure: secure,
    URL: os.Getenv("APP_URL"),
	}

	// create session
	sess := session.Session {
		CookieLifetime: ras.config.cookie.lifetime,
		CookiePersist: ras.config.cookie.persist,
		CookieName: ras.config.cookie.name,
		SessionType: ras.config.sessionType,
		CookieDomain: ras.config.cookie.domain,
	}

	switch ras.config.sessionType {
		case "redis":
			sess.RedisPool = myRedisCache.Conn
		case "mysql", "postgres", "mariadb", "postgresql":
			sess.DBPool = ras.DB.Pool
	}

	ras.Session = sess.InitSession()
	ras.EncryptionKey = os.Getenv("KEY")

	if ras.Debug {
		var views = jet.NewSet(
			jet.NewOSFileSystemLoader(fmt.Sprintf("%s/views", rootPath)),
			jet.InDevelopmentMode(),
		)
	
		ras.JetViews = views
	} else {
		var views = jet.NewSet(
			jet.NewOSFileSystemLoader(fmt.Sprintf("%s/views", rootPath)),
		)
	
		ras.JetViews = views
	}

	ras.createRenderer()

	go ras.Mail.ListenForMail()

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

	if ras.DB.Pool != nil {
		defer ras.DB.Pool.Close()
	}

	if redisPool != nil {
		defer redisPool.Close()
	}

	if badgerConn!= nil {
    defer badgerConn.Close()
  }

	ras.InfoLog.Printf("Listing on port %s", os.Getenv("PORT"))
	err := srv.ListenAndServe()
	ras.ErrorLog.Fatal(err)
}

func (ras *Rasant) checkDotEnv(path string) error { 
	err := ras.CreateFileIfNotExist(fmt.Sprintf("%s/.env", path))
	if err != nil {
		return err
	}
	return nil
}

func (ras *Rasant) createClientRedisCache() *cache.RedisCache {
	cacheClient := cache.RedisCache{
		Conn: ras.createRedisPool(),
		Prefix: ras.config.redis.prefix,
	}

	return &cacheClient
}

func (ras *Rasant) createClientBadgerCache() *cache.BadgerCache {
	cacheClient := cache.BadgerCache{
		Conn: ras.createBadgerConn(),
	}

	return &cacheClient
}

func (ras *Rasant) createRedisPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle: 50,
		MaxActive: 10000,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", ras.config.redis.host, redis.DialPassword(ras.config.redis.password))
		},
		TestOnBorrow: func(conn redis.Conn, t time.Time) error {
			_, err := conn.Do("PING")
      return err
		},
	}
}

func (ras *Rasant) createBadgerConn() *badger.DB {
	db, err := badger.Open(badger.DefaultOptions(ras.RootPath + "/tmp/badger"))
	if err != nil {
    return nil
  }

	return db
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
		Session: ras.Session,
	}

	ras.Render = &myRenderer
}

func (ras *Rasant) createMailer() mailer.Mail {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	m := mailer.Mail{
		Domain: os.Getenv("MAIL_DOMAIN"),
		Templates: ras.RootPath + "/mail",
		Host: os.Getenv("SMTP_HOST"),
		Port: port,
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		Encryption: os.Getenv("SMTP_ENCRYPTION"),
		FromName: os.Getenv("FROM_NAME"),
		FromAddress: os.Getenv("FROM_ADDRESS"),
		Jobs: make(chan mailer.Message, 20),
		Results: make(chan mailer.Result, 20),
		API: os.Getenv("MAILER_API"),
		APIKey: os.Getenv("MAILER_KEY"),
		APIUrl: os.Getenv("MAILER_URL"),
	}	

	return m
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