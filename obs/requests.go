package obs

import (
	"fmt"
	"sync/atomic"
)

var reqCounter atomic.Uint64

func newRequestID() string {
	return fmt.Sprintf("req-%d", reqCounter.Add(1))
}

func (c *Client) SetScene(scene string) (map[string]any, error) {
	return c.Send("SetCurrentProgramScene", map[string]any{
		"sceneName": scene,
	})
}
func (c *Client) GetScene() (string, error) {
	resp, err := c.Send("GetCurrentProgramScene", nil)
	if err != nil {
		return "", err
	}
	name, ok := resp["sceneName"].(string)
	if !ok {
		return "", fmt.Errorf("faild to get name from responce")
	}
	return name, nil
}

func (c *Client) GetActiveSources() (map[string]any, error) {
	scene, err := c.GetScene()
	if err != nil {
		return nil, err
	}
	return c.GetActiveSourcesInScene(scene)
}

func (c *Client) GetActiveSourcesInScene(scene string) (map[string]any, error) {
	return c.Send("GetSceneItemList", map[string]any{
		"sceneName": scene,
	})
}

func (c *Client) GetActiveSourcesInGroup(group string) (map[string]any, error) {
	return c.Send("GetGroupSceneItemList", map[string]any{
		"sceneName": group,
	})
}

func (c *Client) GetSourceID(name string) (int, error) {
	scene, err := c.GetScene()
	if err != nil {
		return -1, err
	}
	return c.GetSourceIDInScene(name, scene)
}

func (c *Client) GetSourceIDInScene(name, scene string) (int, error) {
	sources, err := c.GetActiveSourcesInScene(scene)
	if err != nil {
		return -1, err
	}

	items, ok := sources["sceneItems"].([]any)
	if !ok {
		return -1, fmt.Errorf("obs: unexpectid sceneItems format")
	}

	return seachSceneItemsForName(name, items)
}

func (c *Client) GetSourceIDInGroup(name, group string) (int, error) {
	sources, err := c.GetActiveSourcesInGroup(group)
	if err != nil {
		return -1, err
	}

	items, ok := sources["sceneItems"].([]any)
	if !ok {
		return -1, fmt.Errorf("obs: unexpectid sceneItems format")
	}

	return seachSceneItemsForName(name, items)
}

func seachSceneItemsForName(name string, items []any) (int, error) {
	for _, item := range items {
		source, ok := item.(map[string]any)
		if !ok {
			return -1, fmt.Errorf("obs: unexpectid item format")
		}

		if source["sourceName"] == name {
			id, ok := source["sceneItemId"].(float64)
			if !ok {
				return -1, fmt.Errorf("obs: sceneItemId is not a number")
			}
			return int(id), nil
		}

	}

	return -1, fmt.Errorf("obs: souce %s not found", name)
}

func (c *Client) SetSourceVisability(source string, visable bool) (map[string]any, error) {
	scene, err := c.GetScene()
	if err != nil {
		return nil, err
	}
	return c.SetSourceVisabilityInScene(source, scene, visable)
}
func (c *Client) SetSourceVisabilityInScene(source, scene string, visable bool) (map[string]any, error) {
	id, err := c.GetSourceID(source)
	if err != nil {
		return nil, err
	}

	return c.Send("SetSceneItemEnabled", map[string]any{
		"sceneName":        scene,
		"sceneItemId":      id,
		"sceneItemEnabled": visable,
	})
}

func (c *Client) SetSourceVisabilityInGroup(source, group string, visable bool) (map[string]any, error) {
	id, err := c.GetSourceIDInGroup(source, group)
	if err != nil {
		return nil, err
	}

	return c.Send("SetSceneItemEnabled", map[string]any{
		"sceneName":        group,
		"sceneItemId":      id,
		"sceneItemEnabled": visable,
	})

}

func (c *Client) SetTextSourceText(sourceName, text string) (map[string]any, error) {
	return c.Send("SetInputSettings", map[string]any{
		"inputName": sourceName,
		"inputSettings": map[string]any{
			"text": text,
		},
	})
}
