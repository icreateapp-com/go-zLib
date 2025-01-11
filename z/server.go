package z

import (
	"fmt"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"time"
)

func Serve(setup func(engine *gin.Engine), router func(engine *gin.Engine), middles ...gin.HandlerFunc) {

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
	if err := Config.LoadFile(BasePath(), "config.yml"); err != nil {
		Error.Fatalln(err.Error())
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

	// middleware
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())
	engine.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "OPTIONS,GET,POST,PUT,DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// custom middleware
	engine.Use(middles...)

	// web static directory
	engine.Use(static.Serve("/", static.LocalFile("statics", false)))

	// routers
	router(engine)

	// set trusted proxies
	if _, err := Config.StringSlice("config.trusted_proxies"); err == nil {
		_ = engine.SetTrustedProxies(nil)
	}

	// run setup
	setup(engine)

	// run app
	host, _ := Config.String("config.host")
	port, _ := Config.Int("config.port")

	if err := engine.Run(fmt.Sprintf("%s:%d", host, port)); err != nil {
		Error.Fatalln(err.Error())
	}
}
