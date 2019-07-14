package pcsweb

import (
	"fmt"
	"encoding/json"
	"golang.org/x/net/websocket"
	"net/http"
	"github.com/iikira/BaiduPCS-Go/pcsverbose"
	"time"
)

var routineStatus int
var exitCh = make(chan bool)
var wsSendCh = make(chan routineResponse)
func getChannel() chan routineResponse {
	return wsSendCh
}

func getExitCh() chan bool{
	return exitCh
}

type routineResponse struct{
	typeIndex int
	data Response
}
type pcsConfigJSON struct {
	Name string `json:"name"`
	EnName string `json:"en_name"`
	Value string `json:"value"`
	Desc string `json:"desc"`
}

type pcsOptsJSON struct {
	Name string `json:"name"`
	Value bool `json:"value"`
	Desc string `json:"desc"`
}

type Response struct {
	Code int         `json:"code"`
	Type int         `json:"type"`
	Status int       `json:"status"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func (res *Response) JSON() (data []byte) {
	data, _ = json.Marshal(res)
	return
}
var taskIDD int = 0

func sendResponse(conn *websocket.Conn, rtype int, rstatus int, msg string, data string) (err error){
	response := &Response{
		Code: 0,
		Type: rtype,
		Status: rstatus,
		Msg: msg,
		Data: data,
	}
	if err = websocket.Message.Send(conn, string(response.JSON())); err != nil {
		pcsverbose.Verbosef("websocket send error: %s\n", err.Error())
		return err
	}
	return nil
}

func sendRoutineRequest(typeIndex int , rtype int, rstatus int, msg string, data string){
	wsSendCh := getChannel()
	var senData routineResponse
	senData.typeIndex = typeIndex
	senData.data = Response{0,rtype,rstatus,msg,data}
	wsSendCh <- senData
}
func sendErrorResponse(conn *websocket.Conn, rcode int, msg string) (err error){
	response := &Response{
		Code: rcode,
		Type: 0,
		Status: 0,
		Msg: msg,
		Data: "",
	}
	if err = websocket.Message.Send(conn, string(response.JSON())); err != nil {
		pcsverbose.Verbosef("websocket send error: %s\n", err.Error())
		return err
	}
	return nil
}

func sendHttpErrorResponse(w http.ResponseWriter, rcode int, msg string) {
	response := &Response{
		Code: rcode,
		Type: 0,
		Status: 0,
		Msg: msg,
		Data: "",
	}
	w.Write(response.JSON())
}

func sendHttpResponse(w http.ResponseWriter, msg string, data interface{}) {
	response := &Response{
		Code: 0,
		Type: 0,
		Status: 0,
		Msg: msg,
		Data: data,
	}
	w.Write(response.JSON())
}



func SendWSRequestRoutine()(err error) {
	wsSendCh := getChannel()
	connCh := getConnChannel()
	var getSendData Response
	conn := <- connCh
	//var conn *websocket.Conn
    for{
        select {
			case connNew := <- connCh:{

				conn = connNew;
				// 处理队列
				downList := getDownloadList()

				head := downList.Front()
				for i := 0; i < downList.Len(); i++ {
					e := head.Value.(*dtask)
					fmt.Println("[DEBUG]更新前台正在下载列表",e.ListTask.ID,e.path)
					//MsgBody = fmt.Sprintf("{\"LastID\": %d, \"path\": \"%s\"}", ID, paths[k])
					MsgBody = fmt.Sprintf("{\"LastID\": %d, \"path\": \"%s\"}", e.ListTask.ID, e.path)
								
					getSendData = Response{
						Code: 0,
						Type: 2,
						Status: 1,
						Msg: "添加进任务队列",
						Data: MsgBody,
					}
					if err = websocket.Message.Send(conn, string(getSendData.JSON())); err != nil {
						fmt.Println("\n[DEBUG]conn连接中断")
						pcsverbose.Verbosef("websocket send error: %s\n", err.Error())
						goto SleepFlag
					}

					//MsgBody = fmt.Sprintf("{\"LastID\": %d, \"path\": \"%s\"}", task.ID, task.path)
					MsgBody = fmt.Sprintf("{\"LastID\": %d, \"path\": \"%s\"}", e.ListTask.ID, e.path)
								
					getSendData = Response{
						Code: 0,
						Type: 2,
						Status: 3,
						Msg: "准备下载",
						Data: MsgBody,
					}
					if err = websocket.Message.Send(conn, string(getSendData.JSON())); err != nil {
						fmt.Println("\n[DEBUG]conn连接中断")
						pcsverbose.Verbosef("websocket send error: %s\n", err.Error())
						goto SleepFlag
					}

					if e.complete == true {
						fmt.Println("\n[DEBUG]更新前台完成下载列表",e.ListTask.ID,e.path)
						MsgBody = fmt.Sprintf("{\"LastID\": %d, \"savePath\": \"%s\"}", e.ListTask.ID, e.path)
									
						getSendData = Response{
							Code: 0,
							Type: 2,
							Status: 9,
							Msg: "下载完成",
							Data: MsgBody,
						}
						if err = websocket.Message.Send(conn, string(getSendData.JSON())); err != nil {
							fmt.Println("\n[DEBUG]conn连接中断")
							pcsverbose.Verbosef("websocket send error: %s\n", err.Error())
							goto SleepFlag
						}		
					}
					
					head = head.Next()
				}
				
			}
			default:{
				//fmt.Println("\n[DEBUG]执行任务sendWSRequestRoutine")
				select{
					case getSendData := <- wsSendCh:{

						if err = websocket.Message.Send(conn, string(getSendData.data.JSON())); err != nil {
								fmt.Println("\n[DEBUG]conn连接中断")
								pcsverbose.Verbosef("websocket send error: %s\n", err.Error())
								goto SleepFlag
						}

					}
					default:
				}
			}

					
        }
SleepFlag:
		time.Sleep(time.Duration(1)*time.Second)
	}

}

var (
	NeedPass = -3
	NotLogin = -4
	LoginError = -5
)
