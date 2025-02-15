// Package pcsweb web前端包
package pcsweb

import (
	"fmt"
	"github.com/GeertJohan/go.rice"
	"golang.org/x/net/websocket"
	"html/template"
	"net/http"
)

var distBox *rice.Box
var distMobileBox *rice.Box
	
//创建WS消息发送线程通道
var connCh = make(chan *websocket.Conn)

func getConnChannel() chan *websocket.Conn {
	return connCh
}
// StartServer 开启web服务
func StartServer(port uint, access bool) error {
	if port <= 0 || port > 65535 {
		return fmt.Errorf("invalid port %d", port)
	}

	distBox = rice.MustFindBox("dist") // go.rice 文件盒子
	http.Handle("/dist/", http.StripPrefix("/dist/", http.FileServer(distBox.HTTPBox())))

	distMobileBox = rice.MustFindBox("dist_mobile") // go.rice 文件盒子
	http.Handle("/dist_mobile/", http.StripPrefix("/dist_mobile/", http.FileServer(distMobileBox.HTTPBox())))


	http.HandleFunc("/", rootMiddleware)
	http.HandleFunc("/dist_mobile", middleware(indexMobilePage))
	http.HandleFunc("/index.html", middleware(indexPage))

	http.HandleFunc("/api/v1/login", LoginHandle)
	http.HandleFunc("/api/v1/logout", activeAuthMiddleware(LogoutHandle))
	http.HandleFunc("/api/v1/password", activeAuthMiddleware(PasswordHandle))
	http.HandleFunc("/api/v1/user", activeAuthMiddleware(UserHandle))
	http.HandleFunc("/api/v1/quota", activeAuthMiddleware(QuotaHandle))
	http.HandleFunc("/api/v1/share", activeAuthMiddleware(ShareHandle))
	http.HandleFunc("/api/v1/recycle", activeAuthMiddleware(RecycleHandle))
	http.HandleFunc("/api/v1/download", activeAuthMiddleware(DownloadHandle))
	http.HandleFunc("/api/v1/offline_download", activeAuthMiddleware(OfflineDownloadHandle))
	http.HandleFunc("/api/v1/search", activeAuthMiddleware(SearchHandle))
	http.HandleFunc("/api/v1/setting", activeAuthMiddleware(SettingHandle))
	http.HandleFunc("/api/v1/options", activeAuthMiddleware(OptionsHandle))
	http.HandleFunc("/api/v1/local_file", activeAuthMiddleware(LocalFileHandle))
	http.HandleFunc("/api/v1/file_operation", activeAuthMiddleware(FileOperationHandle))
	http.HandleFunc("/api/v1/mkdir", activeAuthMiddleware(MkdirHandle))
	http.HandleFunc("/api/v1/files", activeAuthMiddleware(fileList))

	http.Handle("/ws", websocket.Handler(WSHandler))
	if access {
		return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	}
	fmt.Println("现在只监听localhost，请注意")
	return http.ListenAndServe(fmt.Sprintf("localhost:%d", port), nil)
}

func indexPage(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	tmpl := boxTmplParse("index", "index.html")
	tmpl.Execute(w, nil)
}

func indexMobilePage(w http.ResponseWriter, r *http.Request) {
	tmpl := template.New("index")
	tmpl.Parse(distMobileBox.MustString("index.html"))
	tmpl.Execute(w, nil)
}