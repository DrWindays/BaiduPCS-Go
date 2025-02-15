package pcsweb

import (
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/iikira/Baidu-Login"
	"github.com/iikira/BaiduPCS-Go/internal/pcsconfig"
	"golang.org/x/net/websocket"
)

func getValueFromWSJson(conn *websocket.Conn, key string) (resString string, err error) {
	var reply string
	if err = websocket.Message.Receive(conn, &reply); err != nil {
		fmt.Println("receive err:", err.Error())
		return
	}
	rJson, err := simplejson.NewJson([]byte(reply))
	if err != nil {
		fmt.Println("create json error:", err.Error())
		return
	}

	resString, _ = rJson.Get(key).String()
	return
}

func WSLogin(conn *websocket.Conn, rJson *simplejson.Json) (err error) {
	var (
		vcode                 string
		vcodestr              string
		BDUSS, PToken, SToken string
	)

	username, _ := rJson.Get("username").String()
	password, _ := rJson.Get("password").String()

	bc := baidulogin.NewBaiduClinet()
	for i := 0; i < 10; i++ {
		lj := bc.BaiduLogin(username, password, vcode, vcodestr)

		switch lj.ErrInfo.No {
		case "0": // 登录成功, 退出循环
			BDUSS, PToken, SToken = lj.Data.BDUSS, lj.Data.PToken, lj.Data.SToken
			goto loginSuccess
		case "400023", "400101": // 需要验证手机或邮箱
			verifyTypes := fmt.Sprintf("[{\"label\": \"mobile %s\", \"value\": \"mobile\"}, {\"label\": \"email %s\", \"value\": \"email\"}]", lj.Data.Phone, lj.Data.Email)
			sendResponse(conn, 1, 2, "需要验证手机或邮箱", verifyTypes)

			verifyType, _ := getValueFromWSJson(conn, "verify_type")

			msg := bc.SendCodeToUser(verifyType, lj.Data.Token) // 发送验证码
			sendResponse(conn, 1, 3, "发送验证码", "")
			fmt.Printf("消息: %s\n\n", msg)

			for et := 0; et < 5; et++ {
				vcode, err = getValueFromWSJson(conn, "verify_code")
				nlj := bc.VerifyCode(verifyType, lj.Data.Token, vcode, lj.Data.U)
				if nlj.ErrInfo.No != "0" {
					errMsg := fmt.Sprintf("{\"error_time\":%d, \"error_msg\":\"%s\"}", et+1, nlj.ErrInfo.Msg)
					sendResponse(conn, 1, 4, "验证码错误", errMsg)
					continue
				}
				// 登录成功
				BDUSS, PToken, SToken = nlj.Data.BDUSS, nlj.Data.PToken, nlj.Data.SToken
				goto loginSuccess
			}
		case "400038": //账号密码错误
			sendResponse(conn, 1, 5, "账号或密码错误", "")
		case "500001", "500002": // 验证码
			if lj.ErrInfo.No == "500002" {
				sendResponse(conn, 1, 4, "验证码错误", "")
			}
			vcodestr = lj.Data.CodeString
			if vcodestr == "" {
				err = fmt.Errorf("未找到codeString")
				return err
			}

			verifyImgURL := "https://wappass.baidu.com/cgi-bin/genimage?" + vcodestr
			sendResponse(conn, 1, 6, verifyImgURL, "")

			vcode, _ = getValueFromWSJson(conn, "verify_code")
			continue
		default:
			err = fmt.Errorf("错误代码: %s, 消息: %s", lj.ErrInfo.No, lj.ErrInfo.Msg)
			sendErrorResponse(conn, -1, err.Error())
			return err
		}
	}

loginSuccess:
	baidu, err := pcsconfig.Config.SetupUserByBDUSS(BDUSS, PToken, SToken)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("百度帐号登录成功:", baidu.Name)
	sendResponse(conn, 1, 7, baidu.Name, "")

	println("globalSessions", GlobalSessions)
	GlobalSessions.WebSocketUnLock(conn.Request())

	err = pcsconfig.Config.Save()
	if err != nil {
		fmt.Printf("保存配置错误: %s\n", err)
	}
	fmt.Printf("保存配置成功\n")

	return err
}

func WSDownload(conn *websocket.Conn, rJson *simplejson.Json) (err error) {
	method, _ := rJson.Get("method").String()

	if method == "download" {
		options := &DownloadOptions{
			IsTest:      false,
			IsOverwrite: true,
		}

		paths, _ := rJson.Get("paths").StringArray()
		dtype, _ := rJson.Get("dtype").String()
		if dtype == "share" {
			options.IsShareDownload = true
		} else if dtype == "locate" {
			options.IsLocateDownload = true
		} else if dtype == "stream" {
			options.IsStreaming = true
		}

		go RunDownload(conn, paths, options)
		return
	}
	return
}

func WSUpload(conn *websocket.Conn, rJson *simplejson.Json) (err error) {
	paths, _ := rJson.Get("paths").StringArray()
	tpath, _ := rJson.Get("tpath").String()

	RunUpload(conn, paths, tpath, nil)
	return
}

func WSHandler(conn *websocket.Conn) {
	fmt.Printf("Websocket新建连接: %s -> %s\n", conn.RemoteAddr().String(), conn.LocalAddr().String())
	connCh = getConnChannel()
	exitCh = getExitCh()
	

	connCh <- conn
	for {
		var reply string

		if err := websocket.Message.Receive(conn, &reply); err != nil {
			fmt.Println("Websocket连接断开:", err.Error())
			conn.Close()
			return
		}


		rJson, err := simplejson.NewJson([]byte(reply))
		if err != nil {
			fmt.Println("receive err:", err.Error())
			return
		}
		rType, _ := rJson.Get("type").Int()

		switch rType {
		case 1:
			WSLogin(conn, rJson)
			if err != nil {
				fmt.Println("WSLogin err:", err.Error())
				continue
			}
		case 2:
			WSDownload(conn, rJson)

			if err != nil {
				fmt.Println("WSDownload err:", err.Error())
				continue
			}
		case 3:
			WSUpload(conn, rJson)
			if err != nil {
				fmt.Println("WSUpload err:", err.Error())
				continue
			}
		}
	}
}
