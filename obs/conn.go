package obs

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
)

type ObsCon struct {
	Name       string
	ObsCon     *websocket.Conn
	Rc         chan string
	WaitingIds []string
	Slack      SlackContainer
}

func NewObsCon(trackName string, slack SlackContainer) *ObsCon {
	obscon := new(ObsCon)

	obscon.Name = trackName
	obscon.Slack = slack

	obscon.Slack.Send2slack(fmt.Sprintf(`:information_source: [%s] obs-controller が起動しました。`, obscon.Name))

	obscon.ObsCon = con()
	obscon.Rc = make(chan string)

	return obscon
}

func con() *websocket.Conn {
	var c *websocket.Conn
	u := url.URL{Scheme: "ws", Host: "127.0.0.1:4444", Path: "/"}
	for {
		var err error
		log.Printf("connecting to %s", u.String())
		c, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			log.Println("dial:", err)
			time.Sleep(time.Second)
			continue
		}
		log.Printf("connected")
		break
	}
	return c
}

func (obscon *ObsCon) Connect() {
	obscon.ObsCon.Close()
	obscon.ObsCon = con()
	obscon.Slack.Send2slack(fmt.Sprintf(`:information_source: [%s] OBS Studio に接続しました。`, obscon.Name))
}

func (obscon *ObsCon) Send(params map[string]string) bool {
	jsonString, err := obscon.SendReceive(params)
	if err != nil {
		log.Println("SendWithParams: SendReceive() returns error")
		return false
	}
	status := gjson.Get(jsonString, "status").String()
	return status == "ok"
}

func (obscon *ObsCon) SendReceive(params map[string]string) (string, error) {
	uuid, _ := uuid.NewUUID()
	params["message-id"] = uuid.String()
	jsonBytes, err := json.Marshal(params)
	if err != nil {
		log.Println("SendWithParams: Invalid params")
		return "", err
	}

	return obscon.SendReceiveCore(uuid, jsonBytes)
}

func (obscon *ObsCon) SendReceiveCore(uuid uuid.UUID, jsonBytes []byte) (string, error) {
	log.Printf("SendReceiveCore: json: %s\n", jsonBytes)

	obscon.WaitingIds = append(obscon.WaitingIds, uuid.String())
	obscon.ObsCon.WriteMessage(websocket.TextMessage, jsonBytes)
	for {
		select {
		case jsonString := <-obscon.Rc:
			resUuid := gjson.Get(jsonString, "message-id").String()
			log.Println("SendReceive: received: ", resUuid)
			if resUuid == uuid.String() {
				return jsonString, nil
			}
		case <-time.After(20 * time.Second):
			log.Println(uuid.String(), "timed out")
			return "", errors.New("timed out")
		}
	}

}

func Receive(obscon *ObsCon, donec chan bool) {
	log.Println("Websocket waiting")
	for {
		select {
		case <-donec:
			log.Println("***** donec received")
			return
		default:
			_, message, err := obscon.ObsCon.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				log.Print("Websocket disconnected. Trying to re-connect.")
				obscon.Slack.Send2slack(fmt.Sprintf(`:bangbang: [%s] OBS Studio から切断されました。再接続します。`, obscon.Name))
				obscon.Connect()
			} else {
				//ray.Ray(string(message))
				log.Printf("recv: %s", message)
				//log.Println("    ids: ", obscon.WaitingIds)
				resUuid := gjson.Get(string(message), "message-id").String()
				//log.Println("    received uuid: ", resUuid)
				for i := 0; i < len(obscon.WaitingIds); i++ {
					//log.Println("        heystack: ", obscon.WaitingIds[i])
					if obscon.WaitingIds[i] == resUuid {
						obscon.Rc <- string(message)
						obscon.WaitingIds = append(obscon.WaitingIds[:i], obscon.WaitingIds[i+1:]...)
						//log.Println("    new ids: ", obscon.WaitingIds)
						i--
					}
				}
			}
		}
	}
}
