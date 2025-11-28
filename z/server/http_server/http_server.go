package http_server

import (
	"fmt"
	"github.com/icreateapp-com/go-zLib/z/server/http_server/http_middleware"
	"os"
	"time"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	. "github.com/icreateapp-com/go-zLib/z"
	"github.com/icreateapp-com/go-zLib/z/provider/trace_provider"
)

func HttpServe(setup func(engine *gin.Engine), router func(engine *gin.Engine), middles ...gin.HandlerFunc) error {

	///////////////////////////////////////////////
	// init system
	///////////////////////////////////////////////

	// set timezone
	timezone := Config.GetString("config.timezone")
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(timezone)
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
	engine.Use(trace_provider.HttpErrorRecoveryMiddleware()) // 错误跟踪中间件
	engine.Use(trace_provider.HttpTraceMiddleware())         // 错误日志中间件
	engine.Use(http_middleware.MobileDetectMiddleware())     // 移动端检测中间件
	engine.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Sec-WebSocket-Protocol, Sec-WebSocket-Extensions")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "OPTIONS,GET,POST,PUT,DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		// WebSocket 特定的 CORS 头部
		if c.GetHeader("Upgrade") == "websocket" {
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Sec-WebSocket-Protocol, Sec-WebSocket-Extensions, Sec-WebSocket-Key, Sec-WebSocket-Version, Upgrade, Connection")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// custom grpc_middleware
	engine.Use(middles...)

	// web static directory
	staticDir := Config.GetString("static_dir", "public")
	engine.Use(static.Serve("/", static.LocalFile(staticDir, false)))

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
