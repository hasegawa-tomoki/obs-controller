package obs

import (
	"fmt"
)

type Source struct {
	Name string
	Path string
}

func PrepareScene(obscon *ObsCon, sceneName string, source Source) bool {
	ok := SetPreviewScene(obscon, sceneName)
	if !ok {
		obscon.Slack.Send2slack(fmt.Sprintf(`:bangbang: [%s] プレビューの "%s" への切り替えに失敗しました`, obscon.Name, sceneName))
		return false
	}
	// obscon.Slack.Send2slack(fmt.Sprintf(`:information_source: プレビューを "%s" に切り替えました`, sceneName))

	if source.Name != "" && source.Path != "" {
		ok := SetSourceVideoLocalFile(obscon, source)
		if ok {
			obscon.Slack.Send2slack(fmt.Sprintf(`:information_source: [%s] ソース "%s" のメディアパスを "%s" に切り替えました`, obscon.Name, source.Name, source.Path))
		} else {
			obscon.Slack.Send2slack(fmt.Sprintf(`:bangbang: [%s] ソースの切替に失敗しました。ソース名: %s  パス: %s`, obscon.Name, source.Name, source.Path))
			return false
		}

	}

	sis := GetSceneItemList(obscon, sceneName)
	return ReloadBrowserSources(obscon, sis)
}

func SwitchScene(obscon *ObsCon, sceneName string) bool {
	ok := SetCurrentScene(obscon, sceneName)
	return ok
}

func ReloadBrowserSources(obscon *ObsCon, sceneItemList SceneItemList) bool {
	for _, si := range sceneItemList.SceneItemss {
		if si.SourceKind == "browser_source" {
			ok := RefreshBrowserSource(obscon, si.SourceName)
			if !ok {
				obscon.Slack.Send2slack(fmt.Sprintf(`:bangbang: [%s] ブラウザ "%s" の再読込に失敗しました`, obscon.Name, si.SourceName))
				return false
			}
			// obscon.Slack.Send2slack(fmt.Sprintf(`:information_source: %s のブラウザ "%s" を再読込しました`, obscon.TrackName, si.SourceName))
		}
	}
	return true
}
