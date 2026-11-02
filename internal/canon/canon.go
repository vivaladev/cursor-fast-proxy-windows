package canon

const (
	Label          = "ru.local.cursor-singbox"
	ListenHost     = "127.0.0.1"
	ListenPort     = 18081
	ServerPort     = 8443
	SNI            = "www.cloudflare.com"
	ProxyURL       = "http://127.0.0.1:18081"
	SingBoxVersion = "1.14.0"
)

var SettingsKeys = []string{
	"http.proxy",
	"http.proxySupport",
	"cursor.general.disableHttp2",
}

var ArgvKeys = []string{
	"proxy-server",
}
