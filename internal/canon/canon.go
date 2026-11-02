package canon

const (
	Label          = "ru.local.cursor-singbox"
	ListenHost     = "127.0.0.1"
	ListenPort     = 18081
	ServerPort     = 8443
	SNI            = "www.cloudflare.com"
	ProxyURL       = "http://127.0.0.1:18081"
	NoProxy        = "localhost,127.0.0.1,::1,.local,.alfabank.ru,.alfaintra.net,.podeli.ru,.core-infra.ru,.platops.ru,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"
	BypassList     = "<local>;*.local;*.alfabank.ru;*.alfaintra.net;*.podeli.ru;*.core-infra.ru;*.platops.ru;10.*;192.168.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*"
	SingBoxVersion = "1.14.0"
)

var SettingsKeys = []string{
	"http.proxy",
	"http.proxySupport",
	"http.noProxy",
	"cursor.general.disableHttp2",
}

var ArgvKeys = []string{
	"proxy-server",
	"proxy-bypass-list",
}
