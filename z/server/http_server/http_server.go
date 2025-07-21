package http_server

import (
	"fmt"
	"os"
	"time"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	. "github.com/icreateapp-com/go-zLib/z"
	"github.com/icreateapp-com/go-zLib/z/server/http_server/http_middleware"
)

func HttpServe(setup func(engine *gin.Engine), router func(engine *gin.Engine), middles ...gin.HandlerFunc) error {

	///////////////////////////////////////////////
	// init system
	///////////////////////////////////////////////

	// set timezone
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST-8", 8*3600)
	}
	time.Local = loc

	// load config
	configFile := "config.yml"
	if envFile := os.Getenv("CONFIG_FILE"); envFile != "" {
		configFile = envFile
	}
	if err := Config.LoadFile(BasePath(), configFile); err != nil {
		panic(err.Error())
	}

	// load config dir
	hasConfigDir, err := IsExists(BasePath("configs"))
	if err != nil {
		panic(err.Error())
	}
	if hasConfigDir {
		if err := Config.LoadDir(BasePath("configs")); err != nil {
			panic(err.Error())
		}
	}

	// Init log system
	debug, _ := Config.Bool("config.debug")
	Log.Init(true, debug)

	// init memory cache
	MemCache.Init(60*time.Minute, 10*time.Minute)

	// validator init
	Validator.Init()

	///////////////////////////////////////////////
	// init http engine
	///////////////////////////////////////////////

	// is production mode
	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// instance engine
	engine := gin.New()

	// grpc_middleware
	engine.Use(gin.Logger())
	engine.Use(http_middleware.ErrorTrackerMiddleware()) // 错误跟踪中间件
	engine.Use(http_middleware.ErrorLogMiddleware())     // 错误日志中间件
	engine.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "OPTIONS,GET,POST,PUT,DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// custom grpc_middleware
	engine.Use(middles...)

	// web static directory
	engine.Use(static.Serve("/", static.LocalFile("statics", false)))

	// routers
	router(engine)

	// set trusted proxies
	if _, err := Config.StringSlice("config.http.trusted_proxies"); err == nil {
		_ = engine.SetTrustedProxies(nil)
	}

	// run setup
	setup(engine)

	// run app
	host := Config.GetString("config.http.host")
	port := Config.GetInt("config.http.port")

	Info.Printf("http server running at %s:%d\n", host, port)

	return engine.Run(fmt.Sprintf("%s:%d", host, port))
}
