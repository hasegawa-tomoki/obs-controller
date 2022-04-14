package obs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/tidwall/gjson"
)

// ↓これで動画切り替えられた
//sendMessage($conn, '{"message-id": "xxxxxxxx",
// "request-type": "SetSourceSettings",
// "sourceName": "movie_本編",
// "sourceSettings": {"local_file": "/Users/tom/Dropbox/アプリ/fortee/phperkaigi-2021/フル/2021-03-28_15-30_Track B_マンガではわからないソフトウェア開発の真理_full.mp4"}}');
// ↓これで直接シーン切り替えできた
//sendMessage($conn, '{"message-id": "xxxxxxxx", "request-type": "SetCurrentScene", "scene-name": "15sec_CM"}');
// ↓これでプレビューシーン切り替えできた
//sendMessage($conn, '{"message-id": "xxxxxxxx", "request-type": "SetPreviewScene", "scene-name": "15sec_CM"}');
//↓ブラウザのURL変更はコレでできた
//sendMessage($conn, '{"message-id": "xxxxxxxx", "request-type": "SetSourceSettings", "sourceName": "web-track-next", "sourceSettings": {"url":"https://fortee.jp/iosdc-japan-2021/scr/track-a-title"}}');
//↓ブラウザリロードこれでできた
//sendMessage($conn, '{"message-id": "xxxxxxxx", "request-type": "RefreshBrowserSource", "sourceName": "web-track-next"}');
//sendMessage($conn, '{"message-id": "xxxxxxxx", "request-type": "RefreshBrowserSource", "sourceName": "web-track-another-2-next"}');

//c.WriteMessage(websocket.TextMessage, []byte(`{"message-id": "xxxxxxxx", "request-type": "GetSourceSettings", "sourceName": "movie_本編"}`))

type SceneItemListItem struct {
	ItemId     int    `json:"itemId"`
	SourceKind string `json:"sourceKind"`
	SourceName string `json:"sourceName"`
	SourceType string `json:"sourceType"`
}

type SceneItemList struct {
	SceneName   string              `json:"sceneName"`
	SceneItemss []SceneItemListItem `json:"sceneItems"`
}

type SceneItem struct {
	Cy              int         `json:"cy"`
	Cx              int         `json:"cx"`
	Alignment       int         `json:"alignment"`
	Name            string      `json:"name"`
	Id              int         `json:"id"`
	Render          bool        `json:"render"`
	Muted           bool        `json:"muted"`
	Locked          bool        `json:"locked"`
	Source_cx       int         `json:"source_cx"`
	Source_cy       int         `json:"source_cy"`
	Type            string      `json:"type"`
	Volume          int         `json:"volume"`
	X               int         `json:"x"`
	Y               int         `json:"y"`
	ParentGroupName string      `json:"parentGroupName"`
	GroupChildren   []SceneItem `json:"groupChildren"`
}

type SceneList struct {
	CurrentScene string      `json:"current-scene"`
	Scenes       []SceneItem `json:"scenes"`
}

type SourceSetting struct {
	Url       string `json:"url"`
	LocalFile string `json:"local_file"`
}

type SourceSettings struct {
	SourceName     string        `json:"sourceName"`
	SourceType     string        `json:"sourceType"`
	SourceSettings SourceSetting `json:"sourceSettings"`
}

type Stats struct {
	Fps                 float64 `json:"fps"`
	RenderTotalFrames   int     `json:"render-total-frames"`
	RenderMissedFrames  int     `json:"render-missed-frames"`
	OutputTotalFrames   int     `json:"output-total-frames"`
	OutputSkippedFrames int     `json:"output-skipped-frames"`
	AverageFrameTime    float64 `json:"average-frame-time"`
	CpuUsage            float64 `json:"cpu-usage"`
	MemoryUsage         float64 `json:"memory-usage"`
	FreeDiskSpace       float64 `json:"free-disk-space"`
}

func SetSourceLocalFile(c *websocket.Conn, sourceName string, fileFullPath string) {
	c.WriteMessage(websocket.TextMessage, []byte(`{"message-id": "xxxxxxxx", "request-type": "GetSourceSettings", "sourceName": "movie_本編"}`))
}

func SetCurrentScene(obscon *ObsCon, sceneName string) bool {
	return obscon.Send(map[string]string{
		"request-type": "SetCurrentScene",
		"scene-name":   sceneName,
	})
}

func SetPreviewScene(obscon *ObsCon, sceneName string) bool {
	return obscon.Send(map[string]string{
		"request-type": "SetPreviewScene",
		"scene-name":   sceneName,
	})
}

func RefreshBrowserSource(obscon *ObsCon, sourceName string) bool {
	return obscon.Send(map[string]string{
		"request-type": "RefreshBrowserSource",
		"sourceName":   sourceName,
	})
}

func GetSceneItemList(obscon *ObsCon, sceneName string) SceneItemList {
	jsonString, _ := obscon.SendReceive(map[string]string{
		"request-type": "GetSceneItemList",
		"sceneName":    sceneName,
	})

	sceneItemList := SceneItemList{}
	json.Unmarshal([]byte(jsonString), &sceneItemList)
	return sceneItemList
}

func GetSceneList(obscon *ObsCon) SceneList {
	jsonString, _ := obscon.SendReceive(map[string]string{
		"request-type": "GetSceneList",
	})

	sceneList := SceneList{}
	json.Unmarshal([]byte(jsonString), &sceneList)
	return sceneList
}

func GetSourceSettings(obscon *ObsCon, sourceName string) SourceSettings {
	jsonString, _ := obscon.SendReceive(map[string]string{
		"request-type": "GetSourceSettings",
		"sourceName":   sourceName,
	})

	sourceSettings := SourceSettings{}
	json.Unmarshal([]byte(jsonString), &sourceSettings)
	return sourceSettings
}

func SetSourceVideoLocalFile(obscon *ObsCon, source Source) bool {
	uuid, _ := uuid.NewUUID()
	jsonString := fmt.Sprintf(
		`{
			"message-id": "%s", 
			"request-type": "SetSourceSettings",
			"sourceName":   "%s",
			"sourceSettings": {
				"local_file": "%s"
			}
		}`,
		uuid.String(),
		source.Name,
		strings.Replace(source.Path, `"`, `\"`, -1),
	)
	jsonString, err := obscon.SendReceiveCore(uuid, []byte(jsonString))

	//log.Println("SetSourceVideoLocalFile: ", jsonString)

	if err != nil {
		return false
	}

	status := gjson.Get(jsonString, "status").String()
	return status == "ok"

	//sendMessage($conn, '{"message-id": "xxxxxxxx",
	// "request-type": "SetSourceSettings",
	// "sourceName": "movie_本編",
	// "sourceSettings": {"local_file": "/Users/tom/Dropbox/アプリ/fortee/phperkaigi-2021/フル/2021-03-28_15-30_Track B_マンガではわからないソフトウェア開発の真理_full.mp4"}}');
}

func GetStats(obscon *ObsCon) Stats {
	jsonString, _ := obscon.SendReceive(map[string]string{
		"request-type": "GetStats",
	})
	bodyJson := gjson.Get(jsonString, "stats").String()

	stats := Stats{}
	json.Unmarshal([]byte(bodyJson), &stats)
	return stats
}
